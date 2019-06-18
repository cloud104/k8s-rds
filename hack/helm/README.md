# KUBE-DB CONTROLLER


## INSTALL

```
helm upgrade kube-db . \
  -f ./values.yaml \
  --namespace=kube-db \
  --set secrets.aws_access_key_id="@TODO" \
  --set secrets.aws_secret_access_key="@TODO" \
  --debug \
  --install
```

## DELETE

```
helm delete --purge kube-db
```
