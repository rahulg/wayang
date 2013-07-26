set -x GOPATH (dirname (status -f))
functions -c fish_prompt __genv_fish_prompt
functions -e fish_prompt
function fish_prompt
	printf "[go:%s] %s" (basename $GOPATH) (__genv_fish_prompt)
end
function gcd
    cd $GOPATH
end
function deactivate
    set -e GOPATH
	functions -c __genv_fish_prompt fish_prompt
	functions -e __genv_fish_prompt
    functions -e deactivate
end
