Docker images
-------------

These images describe a pre-set-up environment for building Please in as
well as various auxiliary images as well.
Currently we only have one build image for Ubuntu which is the canonical
build environment on Linux; in future it'd be nice to add more (e.g. Alpine)
once we can support them.

Note that various dependencies can be seen as more or less optional
(for example there are three Python engines, of which any one is needed
for Please to work). The aspiration is to build all features in each image,
but some may be harder than others if certain packages aren't available.

## Updating images

Once you have made your changes, there are 4 steps to publish them: 
- Ensure your docker daemon is running and you're logged into docker hub
- Run `docker build .` in the directory of the image you want to build. Docker 
  will print out the image ID at the end: `Successfully built ac1817b4fc9e`
- Run `docker tag ac1817b4fc9e thoughtmachine/please_ubuntu:20210720` replacing 
  the image ID with yours, and the tag with todays date in `YYYYMMDD` format. 
  NB. replace `please_ubuntu` based on the image you're building.
- Run `docker push thoughtmachine/please_ubuntu:2021072020` to push to the 
  remote repo, replacing your image and tag to match the above command. Make sure 
  you've done a `docker login` to the docker hub registry. 
