# k8s development Readme

The application ships with a basic k8s config (currently only configured for development) in the `k8s/` directory.  There you will find a `Dockerfile` and yaml configuration to deploy the *reaper* pod, service and ingress.  There is also an example configuration yaml (`k8s-config.yaml`) which needs to be populated by you before skaffold can deploy the s3api.

## install docker desktop and enable kubernetes

* [Install docker desktop](https://www.docker.com/products/docker-desktop)

* [Enable kubernetes on docker desktop](https://docs.docker.com/docker-for-mac/#kubernetes)

## install skaffold

[Install skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold)

## setup ingress controller (do this once on your cluster)

```
kubectl apply -f https://gist.githubusercontent.com/fishnix/a94dd54ec72523024f5a0b99ae7c6e49/raw/013f86ab7af23eb014f25ba18e5d24c4fd329689/traefik-rbac.yaml
kubectl apply -f https://gist.githubusercontent.com/fishnix/a94dd54ec72523024f5a0b99ae7c6e49/raw/ff7fd88c504094e18c470b967f707ad6cd80838e/traefik-ds.yaml
```

## create k8s secret config

* modify the local configuration file in `config/config.json`

* copy example secret yaml `cp k8s/example-k8s-config.yaml k8s/k8s-config.yaml`

* base64 encode the configuration `cat config/config.json | base64 -w0`

* copy output of `config.json` secret into `k8s-config.yaml`

## develop

* run `skaffold dev` in the root of the project

* update your `hosts` file to point spindev.internal.yale.yale.edu to localhost

* use the endpoint `http://<<spindev.internal.yale.edu>>/v1/index`

Saving your code should rebuild and redeploy your project automatically

## [non-]profit
