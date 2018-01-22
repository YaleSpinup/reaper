# reaper

![Reaper](/img/reaper.png?raw=true)

The Reaper reclaims the souls of Spinup TryIT&trade; instances on a set schedule.

## Configuration

The Reaper is configured using the `config/config.json` file.  Start by copying `config.example.json`

### Listen port

`"listen": ":xxxx"`

Configures the listen port.


### Interval

`"interval": "120s"`

Configures how often the reaper runs.


### Log level

`"logLevel": "info"`

Configures how verbose the logging will be.

Valid levels are: `debug`, `info`, `warn`, `error`


### Base URL

`"baseUrl": "http://127.0.0.1:8080/v1/reaper"`

Configures the url for generating renewal links.

Links will be of the format:

`http://127.0.0.1:8080/v1/reaper/renew/i-CcsIuzkwoxbqLFFY?token=JDJhJDEwJFBaU1NYV0JneFFzVG1xUFlrYmlCcC5YSDVidEl6YjRqdE9TZmpybWdiUU93M0x3V05sSlpT`


### Redirect URL

Configures where users will be redirected after they renew an instance from the link.

`"redirectUrl": "https://spinup.internal.yale.edu"`


### Encryption Secret

The encryption secret is used to generate the token for renewal links.  This should be kept safe from prying eyes.

`"encryptionSecret": "super-sekret-token"`


### API Token

This is the API token for non-public/reaper management URLS. 

`"token": "super-er-sekret-token"`


### Search engine

Configures the connection to elasticsearch.  The Reaper uses elasticsearch to find instances that belong in the underworld.

```json
"searchEngine": {
  "endpoint": "http://127.0.0.1:9200"
}
```


### User Datasource

Configures the datasource for user information (first, last, email, etc) used for sending notifications.  Currently this only
supports the `rest` type.

```json
"userDatasource": {
  "type": "rest",
  "endpoint": "http://127.0.0.1:8888/api/v1/users",
  "token": "12345"
}
```

### Email

Configures the email provider details.

```json
"email": {
  "mailserver": "mail.yale.edu",
  "from": "spinup@yale.edu",
  "username": "",
  "password": ""
}
```


### Filter

Filters act as safeguards or limits on the searches done in elasticsearch.  The are converted to keywords and passed to elasticsearch
as `term` queries in the `filter` context.  

For example:
```
  "filter": {
    "foo": "bar",
    "biz": "baz"
  }
```

becomes the following filter in elasticsearch

```
{
  "query": {
    "bool": {
      ...
      "filter": [
       	{ "term"  : { "foo.keyword": "bar" } },
       	{ "term"  : { "biz.keyword": "baz" } }
      ]
    }
  }
}
``` 


### Notifications

When instances reach a certain age, owners are notified that they need to "renew" their instances or they will be reclaimed.  Notifications
are currently done by sending email which is configured in the `email` section above.  The ages are configured in `config.json`:

```json
"notify": {
  "age": [
    "23d",
    "29d"
  ]
}
```


### Decommission

The decommission section configures the decommissioning mechanism.  The reaper `PUT`s the `decom` status to an endpoint.

```json
"decommission": {
  "age": "30d",
  "endpoint": "http://127.0.0.1:8888/v1/servers",
  "token": "12345"
}
```

The actual endpoint will be: `http://127.0.0.1:8888/v1/servers/{{ORG}}/{{INSTANCE_ID}}/status`


### Destroy

The destroy section configures the reaping mechanism.  The reaper `DELETE`s the instance id from an endpoint.

```json
"destroy": {
  "age": "44d",
  "endpoint": "http://127.0.0.1:8888/v1/servers",
  "token": "12345"
}
```

The actual endpoint will be: `http://127.0.0.1:8888/v1/servers/{{ORG}}/{{INSTANCE_ID}}`


### Tagging

The tagging section configures the instance tagging mechanism.  Instance tags are updated when the owners are notified and
instances are renewed.  Tagging is accomplished by `PUT`ing a map of tags to an endpoint.

```json
"tagging": {
  "endpoint": "http://127.0.0.1:8888/v1/servers",
  "token": "12345"
}
```

The actual endpoint will be: `http://127.0.0.1:8888/v1/servers/{{ORG}}/{{INSTANCE_ID}}/tags`


### Event Reporting

Event reporting is supported through the [event reporter library](https://git.yale.edu/spinup/eventreporter).  If no reporters
are configured, no events will be reported.

```json
"eventReporters": {
  "slack": {
    "endpoint": "https://hooks.slack.com/services/xxxxxxxx/xxxxxxxxx/xxxxxxxxxxxx",
    "channel": "#spinup-dev",
    "icon": ":skull_and_crossbones:",
    "username": "Reaper"
  }
}
```


### Full Example

```json
{
  "listen": ":8080",
  "searchEngine": {
    "endpoint": "http://127.0.0.1:9200"
  },
  "filter": {
    "yale:subsidized": "true",
    "yale:org": "fts"
  },
  "notify": {
    "age": [
      "23d",
      "29d"
    ],
    "endpoint": "http://127.0.0.1:8888/v1/notify",
    "token": "12345"
  },
  "decommission": {
    "age": "30d",
    "endpoint": "http://127.0.0.1:8888/v1/servers",
    "token": "12345"
  },
  "destroy": {
    "age": "44d",
    "endpoint": "http://127.0.0.1:8888/v1/destroy",
    "token": "12345"
  },
  "tagging": {
    "endpoint": "http://127.0.0.1:8888/v1/servers",
    "token": "12345"
  },
  "eventReporters": {
    "slack": {
      "endpoint": "https://hooks.slack.com/services/xxxxxxxx/xxxxxxxxx/xxxxxxxxxxxx",
      "channel": "#spinup-dev",
      "icon": ":skull_and_crossbones:",
      "username": "Reaper"
    }
  },
  "interval": "120s",
  "logLevel": "info",
  "baseUrl": "http://127.0.0.1:8080/v1/reaper",  
  "redirectUrl": "https://spinup.internal.yale.edu",
  "encryptionSecret": "super-sekret-token",
  "token": "super-er-sekret-token"
}
```

## Author

E. Camden Fisher <camden.fisher@yale.edu>