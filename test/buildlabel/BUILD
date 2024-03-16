CONFIG.setdefault("TEST_ERRS", [])


def should(msg, a, b):
    err = [f'should|{msg} ("{a}" !== "{b}")']
    if (isinstance(a, str) and not isinstance(b, str)) or (isinstance(a, list) and not isinstance(b, list)) or (isinstance(a, str) and a != b) or (isinstance(a, list) and not all([x == y for x, y in zip(a, b)])):
        # not equal
        CONFIG["TEST_ERRS"] += err


def assert_tests():
    if CONFIG["TEST_ERRS"]:
        fail('\nvvvvvvv\n' + '\n'.join(CONFIG["TEST_ERRS"]) + '\n^^^^^^^\n\n')


bl_f = buildlabel
vis_f = buildlabel

pkg_path = get_base_path()
def_target = basename(pkg_path)
sr_name = subrepo_name()

## TESTS

should("return components in correct position and form", ["///c//a:b|d"], bl_f(path='a', target='b', subrepo='c', subtarget='d', relative=True, aslist=True))
should("return components in correct position and form", ["///c//a:b|d"], bl_f(path='//a', target=':b', subrepo='///c', subtarget='|d', relative=True, aslist=True))

should("return given target from root", "/////:all", bl_f(target='all', relative=True, aslist=False))
should("return given target global", f"///{sr_name}//{pkg_path}:all", bl_f(target='all', relative=False, aslist=False))
should("return current subrepo and package", f"///{sr_name}//{pkg_path}", bl_f(relative=False, aslist=False))
should("return root target", f"/////:all", bl_f(subrepo="///", path="//", target='all', relative=False, aslist=False))
should("return root target", f"/////:all", bl_f(subrepo="/", path="/", target='all', relative=False, aslist=False))

# path resolution

path_append = lambda b, p: f'{b}/{p}' if b else p
sr_c = path_append(sr_name, 'c/c')
pkg_a = path_append(pkg_path, 'a/a')

should("not resolve path when relative=True", f"///../b//../a", bl_f(subrepo='../b', path='../a', relative=True, aslist=False))
should("not resolve path starting with /", "///c//a",             bl_f(path='/a', subrepo='/c', relative=False, aslist=False))
should("resolve path not starting with /", f"///{sr_c}//{pkg_a}", bl_f(path='a/a', subrepo='c/c', relative=False, aslist=False))

pkg_a = path_append(dirname(pkg_path), 'a/c')

should("get parent directory with ../ and remove ./", f"/////{pkg_a}", bl_f(path='../a/b/./../c', subrepo="/", relative=False, aslist=False))
should("get parent directory with ../ and remove ./", f"///a/c//", bl_f(subrepo='a/./b/../c', path="/", relative=False, aslist=False))
should("get parent directory with ../ and remove ./", f"/////a/c", bl_f(path='/a/b/../c/.', relative=False, subrepo="/", aslist=False))

should("return path from root with lots of ../", f"/////a/c", bl_f(path='/../../../../../../../a/b/../c', subrepo="/", relative=False, aslist=False))
should("return path from root with lots of ../", f"/////a/c", bl_f(path='../../../../../../../a/b/../c', subrepo="/", relative=False, aslist=False))
should("return path from root with lots of ../", f"///a/c//", bl_f(subrepo='/../../../../../../../a/b/../c', path="/", relative=False, aslist=False))
should("return path from root with lots of ../", f"///a/c//", bl_f(subrepo='../../../../../../../a/b/../c', path="/", relative=False, aslist=False))

pkg_c = path_append(pkg_path, 'a/c')

should("return path from root with ./", f"/////{pkg_path}", bl_f(path='.', subrepo="/", relative=False, aslist=False))
should("return path from root with ./", f"/////{pkg_c}", bl_f(path='./a/c', subrepo="/", relative=False, aslist=False))
should("return path from root with ./", f"///{sr_c}//", bl_f(subrepo='./c/c', path="/", relative=False, aslist=False))

# defaults

should("default aslist=True", ["///c//a:b|d"], bl_f(path='a', target='b', subrepo='c', subtarget='d', relative=True))
should("default relative=False", "///c//a:b|d", bl_f(path='/a', target='b', subrepo='/c', subtarget='d', aslist=False))
should("default arguments positions label,path,target,subtarget,subrepo", "///d//a:b|c", bl_f('', 'a', 'b', 'c', 'd', relative=True, aslist=False))
should("default label", '///a//c:d|e', bl_f('///a//c:d|e', relative=True, aslist=False))
should("default label", ['///a//c:d|e'], bl_f('///a//c:d|e', relative=True, aslist=True))

# aliases

should("alias path with pkg", "///c//a:b|d", bl_f(pkg='a', target='b', subrepo='c', subtarget='d', relative=True, aslist=False))
should("alias path with package", "///c//a:b|d", bl_f(package='a', target='b', subrepo='c', subtarget='d', relative=True, aslist=False))
should("alias subtarget with out", "///c//a:b|d", bl_f(path='a', target='b', subrepo='c', out='d', relative=True, aslist=False))
should("alias target with name", "///c//a:b|d", bl_f(path='a', name='b', subrepo='c', out='d', relative=True, aslist=False))

# should("fail when two alias are set", "///c//a:b|d", bl_f(path='a', pkg='n', target='b', subrepo='c', out='d', relative=True, aslist=False))
# should("fail when two alias are set", "///c//a:b|d", bl_f(path='a', target='b', subrepo='c', subtarget='e', out='d', relative=True, aslist=False))

# label

should("compose with first argument", "///a//b:c|d", bl_f('///a//b:c', out='d', relative=True, aslist=False))
should("compose with first argument", "///a//b|d", bl_f('///a', path='b', out='d', relative=True, aslist=False))
should("compose with first argument", f"///a//:x|d", bl_f('///a', target='x', out='d', relative=True, aslist=False))
should("compose with first argument", f"///a//{pkg_path}:x|d", bl_f('///a', target='x', out='d', relative=False, aslist=False))
should("compose with first argument", f"/////:x|d", bl_f(':r', target='x', out='d', relative=True, aslist=False))
should("compose with first argument", f"/////r:x|d", bl_f('//r', target='x', out='d', relative=True, aslist=False))

# list

should("accept list on first argument [1] aslist=False", "///a//b:c|d", bl_f(['///a//b:c'], out='d', relative=True, aslist=False))
should("accept list on first argument [1] aslist=True", ["///a//b:c|d"], bl_f(['///a//b:c'], out='d', relative=True, aslist=True))
should("accept list on first argument [2] aslist=False", ["///a//b:c|d", "///a//b:n|d"], bl_f(['///a//b:c', '///a//b:n'], out='d', relative=True, aslist=False))
should("accept list on first argument [2] aslist=True", ["///a//b:c|d", "///a//b:n|d"], bl_f(['///a//b:c', '///a//b:n'], out='d', relative=True, aslist=True))
should("accept list on first argument [0] aslist=False", [], bl_f([], out='d', relative=True, aslist=False))

# vis

should("return PUBLIC for *", ["PUBLIC"], vis_f('*', aslist=True))
should("return PUBLIC for *", "PUBLIC", vis_f('*', aslist=False))
should("keep ... for vis", "/////a/...", vis_f('a/...', relative=True, aslist=False))
should("keep ... for vis", f"/////{pkg_path}/a/...", vis_f('a/...', relative=False, aslist=False))
should("keep ... for vis", f"/////{pkg_path}/a/c/...", vis_f('a/b/../c/...', relative=False, aslist=False))
should("keep ... for vis", "/////a/c/...", vis_f('/a/b/../c/...', relative=False, aslist=False))

assert_tests()

# fail("TEST OK")
