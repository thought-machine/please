"""Lightweight vendorised parts of posixpath / genericpath that we use.

The intention here is to minimise the number of modules we have
to ship, since we only require logical path operations and not
physical ones.

This code remains copyright of the Python authors.
"""

def join_path(a, *p):
    """Join two or more pathname components, inserting '/' as needed.
    If any component is an absolute path, all previous path components
    will be discarded.  An empty last part will result in a path that
    ends with a separator."""
    path = a
    for b in p:
        if b.startswith('/'):
            path = b
        elif path == '' or path.endswith('/'):
            path +=  b
        else:
            path += '/' + b
    return path


def split_path(p):
    """Split a pathname.  Returns tuple "(head, tail)" where "tail" is
    everything after the final slash.  Either part may be empty."""
    i = p.rfind('/') + 1
    head, tail = p[:i], p[i:]
    if head and head != '/'*len(head):
        head = head.rstrip('/')
    return head, tail


def splitext(p, sep='/', extsep='.'):
    """Split the extension from a pathname.

    Extension is everything from the last dot to the end, ignoring
    leading dots.  Returns "(root, ext)"; ext may be empty."""
    sepIndex = p.rfind(sep)
    dotIndex = p.rfind(extsep)
    if dotIndex > sepIndex:
        # skip all leading dots
        filenameIndex = sepIndex + 1
        while filenameIndex < dotIndex:
            if p[filenameIndex] != extsep:
                return p[:dotIndex], p[dotIndex:]
            filenameIndex += 1

    return p, ''


def basename(p):
    """Returns the final component of a pathname"""
    i = p.rfind('/') + 1
    return p[i:]


def dirname(p):
    """Returns the directory component of a pathname"""
    i = p.rfind('/') + 1
    head = p[:i]
    if head and head != '/'*len(head):
        head = head.rstrip('/')
    return head
