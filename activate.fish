set -x GOPATH (dirname (status -f))
function gcd
    cd $GOPATH
end
function deactivate
    set -e GOPATH
    functions -e deactivate
end
