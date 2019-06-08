// Simple Python extension, this happens to be a handy way of testing that
// cc_shared_object actually does something useful.

#include <string>
#include <Python.h>

#include "test/cc_rules/gcc/embedded_files.h"


namespace plz {

PyObject* get_file1(PyObject *self, PyObject *args) {
    return PyUnicode_FromString(embedded_file1_contents().c_str());
}

PyObject* get_file3(PyObject *self, PyObject *args) {
    return PyUnicode_FromString(embedded_file3_contents().c_str());
}

static PyMethodDef so_test_methods[] = {
    {"get_embedded_file_1", get_file1, METH_VARARGS, "gets the first embedded file"},
    {"get_embedded_file_3", get_file3, METH_VARARGS, "gets the third embedded file"},
    {NULL, NULL, 0, NULL}
};

#if PY_MAJOR_VERSION >= 3
#define GETSTATE(m) ((struct module_state*)PyModule_GetState(m))

struct module_state {
    PyObject *error;
};

static int so_test_traverse(PyObject *m, visitproc visit, void *arg) {
    Py_VISIT(GETSTATE(m)->error);
    return 0;
}

static int so_test_clear(PyObject *m) {
    Py_CLEAR(GETSTATE(m)->error);
    return 0;
}

static struct PyModuleDef so_test_def = {
    PyModuleDef_HEAD_INIT,
    "so_test",
    NULL,
    sizeof(struct module_state),
    so_test_methods,
    NULL,
    so_test_traverse,
    so_test_clear,
    NULL
};

#endif  // PY_MAJOR_VERSION >= 3

}  // namespace plz

#if PY_MAJOR_VERSION >= 3
PyMODINIT_FUNC PyInit_so_test() {
    return PyModule_Create(&plz::so_test_def);
}
#else
PyMODINIT_FUNC initso_test() {
    Py_InitModule("so_test", plz::so_test_methods);
}
#endif
