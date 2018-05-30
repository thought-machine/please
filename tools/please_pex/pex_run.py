def run():
    clean_sys_path()
    if not ZIP_SAFE:
        with explode_zip()():
            if MODULE_DIR:
                sys.path = sys.path[:1] + [os.path.join(PEX_PATH, MODULE_DIR.replace('.', '/'))] + sys.path[1:]
            return interact(main)
    else:
        if MODULE_DIR:
            sys.path = sys.path[:1] + [os.path.join(sys.path[0], MODULE_DIR.replace('.', '/'))] + sys.path[1:]
        sys.meta_path.append(SoImport())
        return interact(main)


if __name__ == '__main__':
    if 'PEX_PROFILE_FILENAME' in os.environ:
        with profile(os.environ['PEX_PROFILE_FILENAME'])():
            result = run()
    else:
        result = run()
    sys.exit(result)
