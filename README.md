# permcheck
Read Security Onion roles and permissions and output them in an easy to understand form.

# Install

```
go install github.com/coreyogburn/permcheck
```

# Usage

```
Usage:
  permcheck [OPTIONS] [filter]

Application Options:
  -p, --permissions= Permission file (default: ./permissions)
  -r, --roles=       Role file (default: ./roles)
  -v, --version      Print version and exit

Help Options:
  -h, --help         Show this help message

Arguments:
  filter:            Filter to only a role or permission group

v0.1.3
```

## Output

If no filter is provided, all roles will be printed. First a cummulative list of
permissions will be printed, then each permission group will be printed with the
permissions they grant.

```
> permcheck

agent:
  allPerms: [jobs/process, nodes/write, nodes/read]
  node-admin:
    nodes/write
    nodes/read
  job-processor:
    jobs/process

analyst:
  allPerms: [cases/read, events/ack, events/read, jobs/delete, jobs/pivot, users/read, cases/write, events/write, jobs/write, jobs/read, nodes/read, roles/read]
  node-monitor:
    nodes/read
  user-monitor:
    roles/read
    users/read
  case-admin:
    cases/read
    cases/write
  event-admin:
    events/write
    events/read
    events/ack
  job-admin:
    jobs/delete
    jobs/write
    jobs/read
    jobs/pivot

...
```

If the provided filter is a role, only that role will be printed.

```
> permcheck agent

agent:
  allPerms: [nodes/read, jobs/process, nodes/write]
  job-processor:
    jobs/process
  node-admin:
    nodes/write
    nodes/read
```

If the provided filter is a permission group, the permissions that group
grants will be listed as well as any roles that have that group assigned
to them.

```
> permcheck case-admin

case-admin:
  cases/write
  cases/read

rolesWithPermissionGroup: [analyst, superuser, limited-analyst]
```