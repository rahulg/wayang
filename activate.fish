set -x GOPATH (dirname (status -f))
functions -e __genv_fish_prompt
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
	functions -e fish_prompt
	. ( begin
			printf "function fish_prompt\n\t#"
			functions __genv_fish_prompt
		end | psub )
	functions -e __genv_fish_prompt
    functions -e deactivate
end
