## kp custom-builder delete

Delete a custom builder

### Synopsis

Delete a custom builder in the provided namespace.

namespace defaults to the kubernetes current-context namespace.

```
kp custom-builder delete <name> [flags]
```

### Examples

```
kp cb delete my-builder
kp cb delete -n my-namespace other-builder
```

### Options

```
  -h, --help               help for delete
  -n, --namespace string   kubernetes namespace
```

### SEE ALSO

* [kp custom-builder](kp_custom-builder.md)	 - Custom Builder Commands

###### Auto generated by spf13/cobra on 30-Jul-2020
