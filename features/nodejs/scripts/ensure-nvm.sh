if [ -z "${NVM_DIR}" ] && ! grep -q '/nvm.sh' "$HOME/.bashrc" && ! grep -q '/nvm.sh' "/etc/bash.bashrc" ; then
    echo -e 'export NVM_DIR="REPLACE-ME"\n[ -s "$NVM_DIR/nvm.sh" ] && \\. "$NVM_DIR/nvm.sh"\n[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"' >> "$HOME/.bashrc"
fi