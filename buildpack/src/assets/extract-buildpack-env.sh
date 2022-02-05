# Get a fresh snapshot of environment variables
path_replacer="REPLACE-ME-START:/bin:REPLACE-ME-END"
env -i PATH="${path_replacer}" /cnb/lifecycle/launcher declare +r +f -x > /tmp/cnb_env
# Remove any that are dynamic OS settings
sed -i -r "/declare -x (HOME|SHELL|USER|PWD|OLDPWD|SHLVL|BASH_SUBSHELL|CDPATH|RANDOM|UID|EUID|GROUPS|BASH_VERSION|BASH_VERSINFO|HOSTNAME|HOSTTYPE|OSTYPE|MACHTYPE|IFS|MAILPATH|MAILCHECK|_).*/d" /tmp/cnb_env
# Clear remove declare -x from file lines
sed -i -r "s/declare -x //g" /tmp/cnb_env
# Replace the /bin:REPLACE-ME with the existng path
sed -i -r "s%${path_replacer//%/\\%}%\$\{PATH//\\\/cnb\\\/process:\\\/cnb\\\/lifecycle:/}%g" /tmp/cnb_env
cat /tmp/cnb_env