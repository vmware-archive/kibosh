from test_bind import *
from test_catalog import *
from test_provision import *


def suite():
    s = unittest.TestSuite()

    # instance_id = uuid.uuid4()
    instance_id = "e58a07eb-4d34-4ac7-aa82-d186479a7780"

    TestCatalog.instance_id = instance_id
    s.addTest(TestCatalog('test_catalog'))

    if False:
        test_provision = TestProvision('test_provision')
        test_provision.instance_id = instance_id
        s.addTest(test_provision)

    TestBind.instance_id = instance_id
    s.addTests([
        TestBind('test_bind_response_credentials'),
        TestBind('test_bind_template'),
        TestBind('test_connection')
    ])

    return s


if __name__ == '__main__':
    runner = unittest.TextTestRunner()
    runner.run(suite())
