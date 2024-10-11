
thrift:
	thrift -r -gen go:package_prefix=github.com/abdelaziz-ouhammou/go-impala/services/ interfaces/ImpalaService.thrift
	rm -rf ./services
	mv gen-go services
