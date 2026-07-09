# BastilleBSD API Server

## Commands

The API handles all Bastille commands, and is consistent with
the syntax of the CLI. Any parameters passed via the CLI are named
the same in the API, with some exceptions. Any command that supports
both a jail or a release, will only accept a `target` parameter. See
the `destroy` example below.

## Setup

Run the following command to install the API server.

```shell
git clone https://github.com/BastilleBSD/bastille-api
cd bastille-api
make install
cp /usr/local/etc/bastille-api/config.json.sample /usr/local/etc/bastille-api/config.json
sysrc bastille_api_enable=YES
service bastille-api start
```

The sample config ships with **no API keys**. On first start (with an empty
`apiKeys`), the server generates a random **bootstrap key** with full permissions
and logs it **once** to the service log. Retrieve it right after starting:

```shell
grep "bootstrap key" /var/log/bastille-api/bastille-api.log
# Authorization-ID: bootstrap  API key: <64-hex-character secret>
```

Store that secret somewhere safe — it is never shown again and is not recoverable
from the config (only a salted hash is stored). Use it to authenticate, then
create your own keys through the admin API (see below).

> The server **refuses to start** if the old shipped default key
> (`bastille-api-key`) is present in the config. Remove it and use the bootstrap
> flow instead.

By default the server binds **loopback only** (`127.0.0.1`). For remote access,
run it behind a TLS-terminating reverse proxy (recommended) rather than exposing
the port directly — see `RELEASE_PLAN.md` for proxy configuration. Setting `host`
to a non-loopback address (or `0.0.0.0`) opts into exposing plain HTTP directly.

Requests made via the API must contain an `Authorization: Bearer API_KEY` header as well
as an `Authorization-ID: API_KEY_ID` header.

To use the console feature, you need to `pkg install ttyd`.

## Dependencies

```
bastille
go
ttyd (optional)
```

## API Usage

All requests called via GET will return the supported parameters and options. To actually
run the command, it must be a POST request.

Bastille endpoint: `/api/v1/bastille/command`

Any parameter/option string that has spaces should be passed with either
a `+` or `%20` as the space character. See examples below...

The API supports adding additional keys as well as setting permissions on them. The documentation
at `/swagger/index.html` should have all you need to get started. Keys are stored in
`/usr/local/etc/bastille-api/config.json` (written owner-only, mode `0600`) as a **salted hash** of
the key — never the key itself.

The API key structure has a Key ID (an easy-to-remember name) under which are the `salt`, `hash`
and `permissions`. The Key ID is passed in the `Authorization-ID` header and the actual API key in
the `Authorization` header. Because only the hash is stored, the API cannot recover your key — keep
it safe.

We recommend creating keys through the admin API (authenticate with the bootstrap key, then call
`/api/v1/admin/add`) rather than editing the config by hand. If you do add one manually, the hash is
`sha256(salt + key)`, for example:

```shell
salt="my-random-salt"; key="my-secret-key"
printf "%s" "${salt}${key}" | sha256sum
```

Put `salt` in the `salt` field and the digest in the `hash` field for that Key ID.

## API Examples

The examples below use `https://your-host` — the public name of the reverse
proxy fronting the API. For direct local access on the same machine, use
`http://127.0.0.1:8888` instead. Replace `API_KEY`/`keyid` with your own key (or
`bootstrap` and its secret on a fresh install).

Get supported options and parameters for create
```
curl "https://your-host/api/v1/bastille/create" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Create a jail
```
curl -X POST "https://your-host/api/v1/bastille/create?name=test&release=15.0-release&ip=10.0.0.12&iface=vtnet0" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Create a vnet jail with custom gateway and nameserver
```
curl -X POST "https://your-host/api/v1/bastille/create?name=test&release=15.0-release&ip=10.0.0.12&iface=vtnet0&options=-V+-g+192.168.10.1+-n+192.168.10.1" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Destroy a jail
```
curl -X POST "https://your-host/api/v1/bastille/destroy?target=test&options=-f+-a+-y" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```

Run a command inside a jail
```
curl -X POST "https://your-host/api/v1/bastille/cmd?target=test&command=echo+hi+how%20are%20you" \
     -H "Authorization: Bearer API_KEY" \
     -H "Authorization-ID: keyid"
```
