def start_debugger():
    import debugpy
    debugpy.listen(int(os.environ.get('DEBUG_PORT')))
    debugpy.wait_for_client()
