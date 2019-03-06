# Helm Chart Upgrade Notes

## Mutations
* Introduce a variable
  - No issues
* Remove a key
  - No issues, value disappears from user supplied inputs on `get` command
* Extra key (chart doesn't know what it is)
  - Chart doesn't consume values, but they are stored (`helm get` returns them)
  - If chart upgrade starts to consume this phantom key, that just works (this is mirroring scenario where user chart is old and user supplies key from next version)
* Original install overrides some value, upgrade doesn't re-send the overrides
  - It remembers the override in its collection of user supplied values

Not specific to values, but only a few things can be changed in a pod. Mutations outside of
these things would need some other upgrade path
- `spec.containers[*].image`
- `spec.initContainers[*].image`
- `spec.activeDeadlineSeconds`
- `spec.tolerations`

## Takeaways

Two classes of values:
* Those that can only be set on the initial helm install
* Those than can be continually change

Another learning: it's really easy to build a chart that can't be updated, due
to that first class of value. We accidentally did it several times while testing:
for example, container start command and even environment variables in the spec aren't
mutable.

Charts require _extensive_ testing around upgrades.

# Cycling
### Install
```bash
helm install . --name debug-upgrading
helm upgrade debug-upgrading .
helm install . --name debug-upgrading --set one=override
helm install . --name debug-upgrading --values override.yaml
```

### Inspect
```bash
kubectl logs debug
helm get debug-upgrading

kubectl get secret debug-secret -o yaml
```
