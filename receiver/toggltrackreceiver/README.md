# Toggl Track Receiver

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [development]: logs   |
| Distributions | [] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Areceiver%2Ftoggltrack%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Areceiver%2Ftoggltrack) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Areceiver%2Ftoggltrack%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Areceiver%2Ftoggltrack) |
| Code coverage | [![codecov](https://codecov.io/github/open-telemetry/opentelemetry-collector-contrib/graph/main/badge.svg?component=receiver_toggltrack)](https://app.codecov.io/gh/open-telemetry/opentelemetry-collector-contrib/tree/main/?components%5B0%5D=receiver_toggltrack&displayType=list) |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development
<!-- end autogenerated section -->

This receiver reads timer entries from Toggl Track and turns them into logs.

## Configuration

The following settings are required:

- `api_token:` token used to access Toggl API.

The following settings can be optionally configured:

- `interval` (default = 1m): Specifies the time interval between polls to fetch time entries from the Toggl API.

### Example configurations

Using connection string for authentication:

```yaml
  toggltrack:
    api_token: ${TOGGL_API_TOKEN}
    interval: 30m
```

## Limitations

The receiver is an experiment.

As of today, it comes with several limitations and simplifications.

- **On each collection, the receiver sends all the time entries from Toggl** (last 30 dayes). You need to set up a identify or deduplication to avoid creating duplicates.
- **No data enrichment**. The receiver forwards the IDs (for workspace, project, task) as is with no entrichment (for example, the the project name from the ID). I plan to build enrichment if data analysis in Kibana turns out to be helpful for my "personal observability" project.

## Destinations

Here's how to manage the "time entries as logs" in the destination systems I used for testing.

### Elasticsearch

I built this receiver to collect time entries and send them to Elasticsearch for data visualization and analisys.

Here is how to deal with the current receiver limitations in Elasticsearch.

- To avoid duplicates, I am adding an ingest pipeline to set the time entry `id` field as document `_id` in Elasticsearch.
- To perform basic enrichment, I am setting the project name using specific IDs in my workspace.

I'm sticking to the simplest and quicker solution since this is an experiment I'm not sure I want to move forward.

Here is a sample pipeline to avoid duplicates and perform poor man's enrichment:

```text
{
  "toggl-track-pipeline": {
    "processors": [
      {
        "remove": {
          "field": "_id",
          "ignore_missing": true
        }
      },
      {
        "set": {
          "field": "_id",
          "copy_from": "Attributes.id"
        }
      },
      {
        "set": {
          "field": "Attributes.project_name",
          "value": "Elastic",
          "if": "ctx.Attributes?.project_id == '178435728'"
        }
      },
      {
        "set": {
          "field": "Attributes.project_name",
          "value": "Maintenance",
          "if": "ctx.Attributes?.project_id == '28041930'"
        }
      },
      {
        "set": {
          "field": "Attributes.project_name",
          "value": "Professional",
          "if": "ctx.Attributes?.project_id == '95029662'"
        }
      }
    ]
  }
}
```

And here is an index template to out trigger the pipeline and do basic mapping:

```text
GET _index_template/logs-toggl.track
{
  "index_templates": [
    {
      "name": "logs-toggl.track",
      "index_template": {
        "index_patterns": [
          "logs-toggl.track-*"
        ],
        "template": {
          "settings": {
            "index": {
              "final_pipeline": "toggl-track-pipeline"
            }
          },
          "mappings": {
            "_routing": {
              "required": false
            },
            "numeric_detection": false,
            "dynamic_date_formats": [
              "strict_date_optional_time",
              "yyyy/MM/dd HH:mm:ss Z||yyyy/MM/dd Z"
            ],
            "_meta": {
              "package": {
                "name": "azure"
              },
              "managed_by": "fleet",
              "managed": true
            },
            "dynamic": true,
            "_source": {
              "excludes": [],
              "includes": [],
              "enabled": true
            },
            "dynamic_templates": [],
            "date_detection": true
          }
        },
        "composed_of": [
          "logs@settings",
          "ecs@mappings"
        ],
        "priority": 200,
        "_meta": {
          "package": {
            "name": "toggl"
          }
        },
        "data_stream": {
          "hidden": false,
          "allow_custom_routing": false
        }
      }
    }
  ]
}
```
