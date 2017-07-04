Docker images
-------------

These images describe a pre-set-up environment for building Please in as
well as various auxiliary images as well.
Currently we only have one build imagefor Ubuntu which is the canonical
build environment on Linux; in future it'd be nice to add more (e.g. Alpine)
once we can support them.

Note that various dependencies can be seen as more or less optional
(for example there are three Python engines, of which any one is needed
for Please to work). The aspiration is to build all features in each image,
but some may be harder than others if certain packages aren't available.
