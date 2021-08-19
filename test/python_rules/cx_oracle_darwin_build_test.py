import time
import unittest


class TensorflowTest(unittest.TestCase):

    def test_import(self):
        start = time.time()
        import cx_Oracle
        end = time.time()
        print('Imported tensorflow version %s in %0.2fs' % (cx_Oracle.__version__, end - start))
