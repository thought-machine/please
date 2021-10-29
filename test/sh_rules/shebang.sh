#!/usr/bin/env bash

if [ "`head -1 $0`" != "#!/usr/bin/env bash" ]; then
    echo "Shebang of the original file isn't the first line of the produced artefact" >&2
    exit 1
fi

