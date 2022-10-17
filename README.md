# always

A registry mirror that serves the same image for every tag. Designed to
teach people the importance of pulling by digest.

## Quick start

Run the mirror.

```
$ always ghcr.io/ribbybibby/whalesay
```

Configure your Docker daemon's registry mirror settings. Remember to restart
the daemon.

```
$ cat /etc/docker/daemon.json
{
  "registry-mirrors": ["http://localhost:8080"]
}
```

Try to run the latest `nginx` image.

```
$ docker pull nginx
Using default tag: latest
latest: Pulling from library/nginx
213ec9aee27d: Already exists
42ba3fa28971: Pull complete
9c12cbb3b168: Pull complete
1b07857452e9: Pull complete
d29d1e5fd7b7: Pull complete
Digest: sha256:fb016b2b63fc097c653b812813a37754fffc05ead370f77afa6a86a59aace8bb
Status: Downloaded newer image for nginx:latest
docker.io/library/nginx:latest

$ docker run nginx
 _______
< uh-oh >
 -------
    \
     \
      \
                    ##         .
              ## ## ##        ==
           ## ## ## ## ##    ===
       /"""""""""""""""""\___/ ===
      {                       /  ===-
       \______ O           __/
         \    \         __/
          \____\_______/
```

You can also go direct to the `always` registry.

```
$ docker pull localhost:8080/prom/prometheus
$ docker run localhost:8080/prom/prometheus
 _______
< uh-oh >
 -------
    \
     \
      \
                    ##        .
              ## ## ##       ==
           ## ## ## ##      ===
       /""""""""""""""""___/ ===
  ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
       \______ o          __/
        \    \        __/
          \____\______/
```

## Protect yourself with digests

When you pull manifests by digest, the Docker daemon will refuse a response that
doesn't match the expected digest.

```
Oct 18 10:04:11 foobar dockerd[3045319]: time="2022-10-18T10:04:11.773734008+01:00" level=warning msg="Error persisting manifest" digest="sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1" error="error committing manifest to content store: commit failed: unexpected commit digest sha256:fb016b2b63fc097c653b812813a37754fffc05ead370f77afa6a86a59aace8bb, expected sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1: failed precondition" remote="docker.io/library/nginx@sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1"
```

This makes it impossible for a service like `always` to trick you into running
the wrong thing.

Other tools like `crane` will also reject digest mismatches:

```
$ crane manifest localhost:8080/nginx@sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1
Error: fetching manifest localhost:8080/nginx@sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1: manifest digest: "sha256:fb016b2b63fc097c653b812813a37754fffc05ead370f77afa6a86a59aace8bb" does not match requested digest: "sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1" for "localhost:8080/nginx@sha256:c1c0fedab5e40ba533cbe1f150a49fa3c946ea3fdf4fb4b4cd97ff59930e73d1"
```

## Options

There are some flags you can set to configure the behaviour of `always`.

### Listen address

Change the address the registry listens on with `--listen-address`.

```
$ always ghcr.io/ribbybibby/whalesay --listen-address localhost:12345
```
