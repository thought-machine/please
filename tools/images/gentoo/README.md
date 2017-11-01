Please Gentoo image
-------------------

This image contains a Gentoo system capable of building Please and
its components. Currently it's not fully complete (e.g. does not
include all supported parser engines). It's a non-canonical build
environment but illustrates how to make this work.


Notes
-----

 - Currently we install Python 2.7 and 3.5 since that's what's stable.
   We could use ~amd64 to get 3.6 but it's not really necessary.
 - Currently we're not installing PyPy due to very long build time.
   Later we will probably add it.
