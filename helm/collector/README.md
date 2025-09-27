# Helm

```sh
# deploy using the Elasticsearch exporter
helm template collector helm/collector \        
  --set elasticsearch.endpoints="${ELASTICSEARCH_ENDPOINTS}" \
  --set elasticsearch.username="${ELASTICSEARCH_USERNAME}" \
  --set elasticsearch.password="${ELASTICSEARCH_PASSWORD}" \
  --set wavinsentio.username="${WS_USERNAME}" \
  --set wavinsentio.password="${WS_PASSWORD}" \
  --set zcsazzurro.client_id="${ZCS_CLIENT_ID}" \
  --set zcsazzurro.auth_key="${ZCS_AUTH_KEY}" \
  --set zcsazzurro.thing_key="${ZCS_THING_KEY}" \
  --set toggl.api_token="${TOGGL_API_TOKEN}" | k apply -f -

# deploy using the OTLP exporter
helm template collector helm/collector \
  --set elasticsearch.endpoints="${ELASTICSEARCH_ENDPOINTS}" \
  --set elasticsearch.api_key="${ELASTICSEARCH_API_KEY}" \
  --set wavinsentio.username="${WS_USERNAME}" \
  --set wavinsentio.password="${WS_PASSWORD}" \
  --set zcsazzurro.client_id="${ZCS_CLIENT_ID}" \
  --set zcsazzurro.auth_key="${ZCS_AUTH_KEY}" \
  --set zcsazzurro.thing_key="${ZCS_THING_KEY}" \
  --set toggl.api_token="${TOGGL_API_TOKEN}" | k apply -f -
```
