Docker images
-------------

These images are used for Please's CI/CD pipeline; they also serve as examples of how to set up
containers which Please can run in.

The canonical home for these images is Github Container Registry:

* https://ghcr.io/thought-machine/please_alpine
* https://ghcr.io/thought-machine/please_freebsd_builder
* https://ghcr.io/thought-machine/please_ubuntu
* https://ghcr.io/thought-machine/please_ubuntu_alt

## Updating images

Once you have made your changes, ensure that your local docker daemon is running and you are
[logged in to Github Container Registry][1], and then run `./tools/images/build.sh`.

[1]: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry
