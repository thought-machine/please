#include <fst/vector-fst.h>
#include "src/build/cc/fst_lib.h"

namespace thought_machine {

std::string VectorFstType() {
    typedef fst::VectorFst<fst::StdArc> VectorFst;
    auto transducer = VectorFst();
    return transducer.Type();
}

}  // namespace thought_machine
