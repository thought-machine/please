Please Ubuntu image
-------------------

This image contains everything needed to build Please and all its
parser engines and run all the additional tests. It's the canonical
Linux build environment for Please.

The only tests that are excluded are tests that run in Docker since
it's not easily possible to spawn more Docker containers from inside
a container.

Notes
-----

 - We download a precompiled protoc to avoid trying to compile the whole
   thing from scratch. A sufficiently recent packaged version would be
   preferable.
 - libunittest++-dev is available in Ubuntu but the version is fairly old
   and core dumps on several tests. Installing v2.0.0 ourselves fixes this,
   although the naming is a bit inconsistent (tracked in #200). Regardless,
   we fix that up with a couple of judicious symlinks.
