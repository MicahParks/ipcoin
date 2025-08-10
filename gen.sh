rm -rf ./proto_out # buf generate doesn't like this.
rm -rf ./proto/*.pb.go # When files are renamed, remove the old ones.
buf generate
buf export . --output ./proto_out # So GoLand can resolve dependencies.
rm $(ls proto/*.swagger.json | grep -v ip_coin_service.swagger.json)
