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

Authentication is easiest done with a classic personal access token with the `write:packages` and
`read:packages` scope. Create this at <https://github.com/settings/tokens>, and then use it as the
password for `docker login ghcr.io -u YOUR_GITHUB_USERNAME`. You must be a repository admin to
push the images.

[1]: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry
