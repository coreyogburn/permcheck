package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	col "github.com/coreyog/generics/collections"
	"github.com/jessevdk/go-flags"
	"github.com/samber/lo"
)

type Positional struct {
	Filter string `positional-arg-name:"filter" description:"Filter to only a role or title"`
}

type Arguments struct {
	PermissionsFile string `short:"p" long:"permissions" description:"Permission file" required:"true"`
	RolesFile       string `short:"r" long:"roles" description:"Role file" required:"true"`
	Positional      `positional-args:"true"`
}

var args *Arguments

func main() {
	args = &Arguments{}
	_, err := flags.Parse(args)
	if err != nil {
		if flags.WroteHelp(err) {
			return
		}

		panic(err)
	}

	rolesToTitles := map[string]*col.Set[string]{}      // map[role][]title, i.e. superuser => [admin, moderator]
	titleToPermissions := map[string]*col.Set[string]{} // map[title][]permission, i.e. admin => [events/write, grid/read]
	rolesToPermissions := map[string]*col.Set[string]{} // map[role][]permission, i.e. superuser => [events/write, grid/read]

	// map[permission][]titles that have that permission
	permissionToTitle, err := mapFile(args.PermissionsFile)
	if err != nil {
		panic(err)
	}

	titleToRole, err := mapFile(args.RolesFile)
	if err != nil {
		panic(err)
	}

	titleInheritence := map[string]*col.Set[string]{}

	for k, v := range permissionToTitle {
		if !strings.Contains(k, "/") {
			if titleInheritence[k] == nil {
				titleInheritence[k] = &col.Set[string]{}
			}
			titleInheritence[k].Add(v...)
			delete(permissionToTitle, k)
		} else {
			for _, title := range v {
				if titleToPermissions[title] == nil {
					titleToPermissions[title] = &col.Set[string]{}
				}

				titleToPermissions[title].Add(k)
			}
		}
	}

	for k, v := range titleToRole {
		for _, role := range v {
			if rolesToTitles[role] == nil {
				rolesToTitles[role] = &col.Set[string]{}
			}

			rolesToTitles[role].Add(k)
		}
	}

	for k, v := range titleInheritence {
		for _, title := range v.Slice() {
			for _, perm := range titleToPermissions[k].Slice() {
				if titleToPermissions[title] == nil {
					titleToPermissions[title] = &col.Set[string]{}
				}

				titleToPermissions[title].Add(perm)
			}
		}
	}

	for role, titles := range rolesToTitles {
		for _, title := range titles.Slice() {
			for _, perm := range titleToPermissions[title].Slice() {
				if rolesToPermissions[role] == nil {
					rolesToPermissions[role] = &col.Set[string]{}
				}

				rolesToPermissions[role].Add(perm)
			}
		}
	}

	roles := lo.Keys(rolesToTitles)
	titles := lo.Keys(titleToPermissions)

	var filterIsRole, filterIsTitle bool

	if args.Filter != "" {
		filterIsRole = lo.Contains(roles, args.Filter)
		filterIsTitle = lo.Contains(titles, args.Filter)
	}

	if !filterIsTitle {
		for role, titles := range rolesToTitles {
			if !filterIsRole || role == args.Filter {
				fmt.Printf("%s:\n", role)
				allPerms := col.Set[string]{}
				buf := bytes.NewBuffer([]byte{})
				for _, title := range titles.Slice() {
					fmt.Fprintf(buf, "  %s:\n", title)
					allPerms.Add(titleToPermissions[title].Slice()...)
					for _, perm := range titleToPermissions[title].Slice() {
						fmt.Fprintf(buf, "    %s\n", perm)
					}
				}
				fmt.Printf("  allPerms: [%s]\n", strings.Join(allPerms.Slice(), ", "))
				fmt.Println(buf.String())
			}
		}
	} else {
		for title, perms := range titleToPermissions {
			if !filterIsTitle || title == args.Filter {
				fmt.Printf("%s:\n", title)
				for _, perm := range perms.Slice() {
					fmt.Printf("  %s\n", perm)
				}
			}
		}

		if filterIsTitle {
			rolesWithTitle := []string{}
			for role, titles := range rolesToTitles {
				if titles.InSet(args.Filter) {
					rolesWithTitle = append(rolesWithTitle, role)
				}
			}

			fmt.Printf("\nrolesWithTitle: [%s]\n", strings.Join(rolesWithTitle, ", "))
		}
	}
}

func mapFile(path string) (map[string][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(raw), "\n")
	m := make(map[string][]string)
	for _, line := range lines {
		hashLoc := strings.Index(line, "#")
		if hashLoc != -1 {
			// remove comments
			line = line[:hashLoc]
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			// no colon, bad line
			continue
		}

		key := strings.TrimSpace(parts[0])
		values := strings.Split(strings.TrimSpace(parts[1]), " ")

		m[key] = lo.Map(values, func(v string, i int) string {
			return strings.TrimSpace(v)
		})
	}

	return m, nil
}
