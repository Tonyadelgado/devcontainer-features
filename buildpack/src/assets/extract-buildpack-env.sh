env -i PATH=/bin:REPLACE-ME /cnb/lifecycle/launcher declare +r +f -x > /tmp/cnb_env
sed -i -r "/declare -x (HOME|SHELL|USER|PWD|OLDPWD|SHLVL|BASH_SUBSHELL|CDPATH|RANDOM|UID|EUID|GROUPS|BASH_VERSION|BASH_VERSINFO|HOSTNAME|HOSTTYPE|OSTYPE|MACHTYPE|IFS|MAILPATH|MAILCHECK|_).*/d" /tmp/cnb_env
sed -i -r "s/declare -x //g" /tmp/cnb_env
sed -i -r "s/\/bin:REPLACE-ME/\$\{PATH\}/g" /tmp/cnb_env
cat /tmp/cnb_env