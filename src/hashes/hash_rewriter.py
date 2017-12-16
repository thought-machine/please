"""A Python script to rewrite hashes in BUILD files."""

from _ast import Expr, Call, Str, List, PyCF_ONLY_AST


# These are templated in by Go. It's a bit hacky but is a way of avoiding
# passing arbitrary arguments through Go / C calls.
FILENAME = '__FILENAME__'
TARGETS = {__TARGETS__}
PLATFORM = '__PLATFORM__'


def get_node_name(node):
    """Returns the name of a node if it's a target that we're interested in."""
    if isinstance(node, Expr) and isinstance(node.value, Call):
        for keyword in node.value.keywords:
            if keyword.arg == 'name':
                if isinstance(keyword.value, Str) and keyword.value.s in TARGETS:
                    return keyword.value.s


def replace_hash(line, before, after):
    """Rewrites a hash within one particular line. Returns updated line."""
    quote = lambda s, q: q + s + q
    return line.replace(quote(before, '"'), quote(after, '"')).replace(quote(before, "'"), quote(after, "'"))


with _open(FILENAME) as f:
    lines = f.readlines()
    tree = _compile(''.join(lines), FILENAME, 'exec', PyCF_ONLY_AST)

for node in iter_child_nodes(tree):
    name = get_node_name(node)
    if name:
        for keyword in node.value.keywords:
            if keyword.arg == 'hashes' and isinstance(keyword.value, List):
                # lineno - 1 because lines in the ast are 1-indexed
                candidates = {dep.s: dep.lineno - 1 for dep in keyword.value.elts
                              if isinstance(dep, Str)}
                # Filter by any leading platform (i.e. linux_amd64: abcdef12345).
                platform_candidates = {k: v for k, v in candidates.items() if PLATFORM in k}
                prefix = ''
                if len(platform_candidates) == 1:
                    candidates = platform_candidates
                    prefix = PLATFORM + ': '
                # Should really do something here about multiple hashes and working out which
                # is which...
                current, lineno = candidates.popitem()
                prefix, colon, _ = current.rpartition(':')
                if colon:
                    colon += ' '
                lines[lineno] = replace_hash(lines[lineno], current, prefix + colon + TARGETS[name])
            elif keyword.arg == 'hash' and isinstance(keyword.value, Str):
                lineno = keyword.value.lineno - 1
                current = keyword.value.s
                prefix = current[:current.find(':') + 2] if ': ' in current else ''
                lines[lineno] = replace_hash(lines[lineno], current, prefix + TARGETS[name])


with _open(FILENAME, 'w') as f:
    for line in lines:
        f.write(line)
