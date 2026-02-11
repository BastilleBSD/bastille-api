# BastilleBSD API + UI

## Commands

The API handles all bastille commands, and is consistent with
the syntax of the CLI. Any parameter passed via the CLI is named
the same in the API, with some exceptions. Any command that supports
both a jail or a release, will only accept a `target` parameter. See
the `destroy` example below.

Ir also handles all Rocinante commands, except for the `CP` hook.

## Setup

You can either build and run, or just run the package.

```shell
go build
./bastille-ui
```
or
```
go run .
```
To run only the API: `go run . --api-only`

To run in debug mode: `go run . --debug`

API config file: `api/config.json`

WebUI config file: `web/config.json`

Requests made via the API must contain an `Authorization: Bearer API_KEY` header as well
as an `Authorization-ID: keyid` header. The `API_KEY` can be set inside
the `api/config.json` file. The `API_KEY` is set every time the program starts.

For the WebUI, the `web/config.json` file contains a default username and password to
log in. Simply visit http://host:port to get started.

To use the console on the homepage, you need to `pkg install ttyd`.

## Dependencies

```
bastille
rocinante (optional)
go
ttyd
```

## API Usage

All requests called via GET will return the supported parameters and options. To actually
run the command, it must be a POST request.

Bastille endpoint: `/api/v1/bastille/command`

Rocinante endpoint: `/api/v1/rocinante/command`

Any parameter/option string that has spaces should be passed with either
a `+` or `%20` as the space character. See examples below...

The API supports adding additional keys as well as setting permissions on them. The documentation
at /swagger/index.html should have all you need to get started. Keys are stored in `api/config.json`
as the hashed value of the specified key. The initial hash in the sample file comes from the following
command, `printf "my-random-saltbastille-api-key" | sha256sum`.

The `my-random-salt` above should go in the `salt` json parameter, while the output of the above
command should go into the `hash` parameter. The actual name of the key (keyid) can be anything
you want it to be, but that is the value that goes into the `Authorization-ID` header.

The API key structure has a keyID (easy to remember name), under which are the 
salt, hash and permissions. The key ID must be passed under the `Authorization-ID` header
and the actual API key must be passed with the `Authorization` header.The API has no way
of remembering or storing your actual API key, so keep it safe.

You can add keys manually, but we recommend adding them through the API interface.

## API Examples

Get supported options and parameters for create
```
curl "http://ip:port/api/v1/bastille/create" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Create a jail
```
curl -X POST "http://ip:port/api/v1/bastille/create?name=test&release=15.0-release&ip=10.0.0.12&iface=vtnet0" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Create a vnet jail with custom gateway and nameserver
```
curl -X POST "http://ip:port/api/v1/bastille/create?name=test&release=15.0-release&ip=10.0.0.12&iface=vtnet0&options=-V+-g+192.168.10.1+-n+192.168.10.1" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Destroy a jail
```
curl -X POST "http://ip:port/api/v1/bastille/destroy?target=test&options=-f+-a+-y" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Run a command inside a jail
```
curl -X POST "http://ip:port/api/v1/bastille/cmd?target=test&command=echo+hi+how%20are%20you" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Apply a template using rocinante
```
curl -X POST "http://ip:port/api/v1/rocinante/template?template=custom/template" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```
