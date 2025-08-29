# Test Resources Setup

## Prerequisites

Complete the initial GKE setup from the [main README](../../../README.md) first, then initialize the test resources overlay:

```bash
task config:init
```

## 1. Choose Your Test Resources

Your directory includes ready-to-use resources:
- **postgresql.yaml** - PostgreSQL + Database 
- **kafka.yaml** - Kafka + Topic

To test only specific resources, comment out what you don't need in `kustomization.yaml`:
```yaml
resources:
  - postgresql.yaml    # PostgreSQL + Database
  # - kafka.yaml       # Comment out if not needed
```

## 2. Add Custom Resources (Optional)

Add your own resource files and reference them:
```yaml
resources:
  - postgresql.yaml
  - my-custom-service.yaml
```

## 3. Deploy and Test

```bash
# Deploy your resources
task resources:deploy

# Check status
task resources:status

# View operator logs
task logs
```

## 4. Cleanup

```bash
# Remove your test resources (keeps operator running)
task resources:destroy

# Or destroy everything (cluster + operator)
task addon:destroy
task cluster:destroy
```

## What You Get

- **Name prefixing**: All resources get `yourname-` prefix to avoid conflicts
- **Auto-labeling**: Resources tagged with `developer=yourname` for easy management
- **Project injection**: Your Aiven project name is automatically added
