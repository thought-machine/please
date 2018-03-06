def run():
    if MODULE_DIR:
        override_import(MODULE_DIR)
        sys.meta_path.append(SoImport())
    clean_sys_path()
    if not ZIP_SAFE:
        with explode_zip()():
            return interact(main)
    else:
        return interact(main)


if __name__ == '__main__':
    if 'PEX_PROFILE_FILENAME' in os.environ:
        with profile(os.environ['PEX_PROFILE_FILENAME'])():
            result = run()
    else:
        result = run()
    sys.exit(result)
