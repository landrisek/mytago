export GOPATH=/usr/lib/go-1.6/src
installPackage() {
	if [ ! -d $GOPATH/$@ ]; then
		go get $@	
	fi
}
installPackage github.com/jinzhu/configor
installPackage github.com/BurntSushi/toml
installPackage github.com/go-yaml/yaml/tree/v1
installPackage github.com/go-sql-driver/mysql