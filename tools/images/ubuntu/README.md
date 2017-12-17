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

 - The currently recommended version is Artful (17.10). A few things
   work notably better and/or easier in this version, although it's certainly
   possibly to get things working in Xenial as well with a bit more work.
   Most notably you'll need to build a version of unittest++ yourself
   if you want to run the C++ tests.
