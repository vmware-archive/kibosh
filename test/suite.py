import unittest

from test_catalog import TestCatalog
from test_provision import TestProvision


def suite():
    suite = unittest.TestSuite()
    suite.addTest(TestCatalog('test_catalog'))
    suite.addTest(TestProvision('test_provision'))
    return suite


if __name__ == '__main__':
    runner = unittest.TextTestRunner()
    runner.run(suite())
