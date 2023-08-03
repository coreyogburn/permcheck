package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	col "github.com/coreyog/generics/collections"
	"github.com/jessevdk/go-flags"
	"github.com/samber/lo"
)

type Positional struct {
	Filter string `positional-arg-name:"filter" description:"Filter to only a role or permission group"`
}

type Arguments struct {
	PermissionsFile string `short:"p" long:"permissions" description:"Permission file" default:"./permissions"`
	RolesFile       string `short:"r" long:"roles" description:"Role file" default:"./roles"`
	PrintVersion    bool   `short:"v" long:"version" description:"Print version and exit"`
	Positional      `positional-args:"true"`
}

var args *Arguments

//go:embed VERSION
var version string

func main() {
	args = &Arguments{}
	_, err := flags.Parse(args)
	if err != nil {
		if flags.WroteHelp(err) {
			fmt.Println(version)
			return
		}

		panic(err)
	}

	if args.PrintVersion {
		fmt.Println(version)
		return
	}

	rolesToPGroups := map[string]*col.Set[string]{}       // map[role][]pgroup, i.e. superuser => [admin, moderator]
	pgroupsToPermissions := map[string]*col.Set[string]{} // map[pgroup][]permission, i.e. admin => [events/write, grid/read]
	rolesToPermissions := map[string]*col.Set[string]{}   // map[role][]permission, i.e. superuser => [events/write, grid/read]

	// map[permission][]pgroups that have that permission
	permissionToPGroup, err := mapFile(args.PermissionsFile)
	if err != nil {
		panic(err)
	}

	pgroupToRole, err := mapFile(args.RolesFile)
	if err != nil {
		panic(err)
	}

	pgroupInheritence := map[string]*col.Set[string]{}

	for k, v := range permissionToPGroup {
		if !strings.Contains(k, "/") {
			if pgroupInheritence[k] == nil {
				pgroupInheritence[k] = &col.Set[string]{}
			}
			pgroupInheritence[k].Add(v...)
			delete(permissionToPGroup, k)
		} else {
			for _, pg := range v {
				if pgroupsToPermissions[pg] == nil {
					pgroupsToPermissions[pg] = &col.Set[string]{}
				}

				pgroupsToPermissions[pg].Add(k)
			}
		}
	}

	for k, v := range pgroupToRole {
		for _, role := range v {
			if rolesToPGroups[role] == nil {
				rolesToPGroups[role] = &col.Set[string]{}
			}

			rolesToPGroups[role].Add(k)
		}
	}

	for k, v := range pgroupInheritence {
		for _, pg := range v.Slice() {
			for _, perm := range pgroupsToPermissions[k].Slice() {
				if pgroupsToPermissions[pg] == nil {
					pgroupsToPermissions[pg] = &col.Set[string]{}
				}

				pgroupsToPermissions[pg].Add(perm)
			}
		}
	}

	for role, pgroups := range rolesToPGroups {
		for _, pg := range pgroups.Slice() {
			for _, perm := range pgroupsToPermissions[pg].Slice() {
				if rolesToPermissions[role] == nil {
					rolesToPermissions[role] = &col.Set[string]{}
				}

				rolesToPermissions[role].Add(perm)
			}
		}
	}

	roles := lo.Keys(rolesToPGroups)
	groups := lo.Keys(pgroupsToPermissions)

	var filterIsRole, filterIsPGroup bool

	if args.Filter != "" {
		filterIsRole = lo.Contains(roles, args.Filter)
		filterIsPGroup = lo.Contains(groups, args.Filter)
	}

	if !filterIsPGroup {
		for role, pgroups := range rolesToPGroups {
			if !filterIsRole || role == args.Filter {
				fmt.Printf("%s:\n", role)
				allPerms := col.Set[string]{}
				buf := bytes.NewBuffer([]byte{})
				for _, pg := range pgroups.Slice() {
					fmt.Fprintf(buf, "  %s:\n", pg)
					allPerms.Add(pgroupsToPermissions[pg].Slice()...)
					for _, perm := range pgroupsToPermissions[pg].Slice() {
						fmt.Fprintf(buf, "    %s\n", perm)
					}
				}
				fmt.Printf("  allPerms: [%s]\n", strings.Join(allPerms.Slice(), ", "))
				fmt.Println(buf.String())
			}
		}
	} else {
		for pg, perms := range pgroupsToPermissions {
			if !filterIsPGroup || pg == args.Filter {
				fmt.Printf("%s:\n", pg)
				for _, perm := range perms.Slice() {
					fmt.Printf("  %s\n", perm)
				}
			}
		}

		if filterIsPGroup {
			rolesWithPGroup := []string{}
			for role, pgroups := range rolesToPGroups {
				if pgroups.InSet(args.Filter) {
					rolesWithPGroup = append(rolesWithPGroup, role)
				}
			}

			fmt.Printf("\nrolesWithPermissionGroup: [%s]\n", strings.Join(rolesWithPGroup, ", "))
		}
	}
}

func mapFile(path string) (map[string][]string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}

		path = filepath.Join(home, path[1:])
	}

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
