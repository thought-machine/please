import time
import unittest


class TensorflowTest(unittest.TestCase):

    def test_import(self):
        start = time.time()
        import tensorflow
        end = time.time()
        print('Imported tensorflow version %s in %0.2fs' % (tensorflow.__version__, end - start))
