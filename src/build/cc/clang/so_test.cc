// Simple Python extension, this happens to be a handy way of testing that
// cc_shared_object actually does something useful.

#include <string>
#include <Python.h>

#include "src/build/cc/clang/embedded_files.h"


namespace thought_machine {

PyObject* get_file1(PyObject *self, PyObject *args) {
    return PyString_FromString(embedded_file1_contents().c_str());
}

PyObject* get_file3(PyObject *self, PyObject *args) {
    return PyString_FromString(embedded_file3_contents().c_str());
}

static PyMethodDef module_methods[] = {
    {"get_embedded_file_1", get_file1, METH_VARARGS, "gets the first embedded file"},
    {"get_embedded_file_3", get_file3, METH_VARARGS, "gets the third embedded file"},
    {NULL, NULL, 0, NULL}
};

}  // namespace thought_machine

PyMODINIT_FUNC initso_test() {
    Py_InitModule("so_test", thought_machine::module_methods);
}
