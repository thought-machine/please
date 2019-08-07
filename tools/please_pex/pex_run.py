def add_module_dirs_to_sys_path(dirnames):
    """Adds the given dirname to sys.path if it's nonempty."""
    for dirname in dirnames:
        if dirname:
            sys.path = sys.path[:1] + [os.path.join(sys.path[0], dirname)] + sys.path[1:]
            sys.meta_path.insert(0, ModuleDirImport(dirname))


def run():
    if not ZIP_SAFE:
        with explode_zip()():
            add_module_dirs_to_sys_path(MODULE_DIRS)
            return interact(main)
    else:
        add_module_dirs_to_sys_path(MODULE_DIRS)
        sys.meta_path.append(SoImport())
        return interact(main)


if __name__ == '__main__':
    if 'PEX_PROFILE_FILENAME' in os.environ:
        with profile(os.environ['PEX_PROFILE_FILENAME'])():
            result = run()
    else:
        result = run()
    sys.exit(result)
