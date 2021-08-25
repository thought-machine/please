# This library simulates a regular python library, to see
# if the transitive dependencies are also correctly resolved.

import certifi


def get_certifi_module():
    return certifi

