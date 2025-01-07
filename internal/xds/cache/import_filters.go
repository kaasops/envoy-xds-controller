/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// for work run command - 'go generate ./pkg/xds/cache'

package cache

//go:generate sh -c "echo '/*' > import_filters.gen.go"
//go:generate sh -c "echo 'Copyright 2024.' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo 'Licensed under the Apache License, Version 2.0 (the \"License\");' >> import_filters.gen.go"
//go:generate sh -c "echo 'you may not use this file except in compliance with the License.' >> import_filters.gen.go"
//go:generate sh -c "echo 'You may obtain a copy of the License at' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo '    http://www.apache.org/licenses/LICENSE-2.0' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo 'Unless required by applicable law or agreed to in writing, software' >> import_filters.gen.go"
//go:generate sh -c "echo 'distributed under the License is distributed on an \"AS IS\" BASIS,' >> import_filters.gen.go"
//go:generate sh -c "echo 'WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.' >> import_filters.gen.go"
//go:generate sh -c "echo 'See the License for the specific language governing permissions and' >> import_filters.gen.go"
//go:generate sh -c "echo 'limitations under the License.' >> import_filters.gen.go"
//go:generate sh -c "echo '*/' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo '//  GENERATED FILE -- DO NOT EDIT' >> import_filters.gen.go"
//go:generate sh -c "echo '//  For more info read - https://github.com/envoyproxy/go-control-plane/issues/390' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo 'package cache' >> import_filters.gen.go"
//go:generate sh -c "echo '' >> import_filters.gen.go"
//go:generate sh -c "echo 'import (' >> import_filters.gen.go"
//go:generate sh -c "go list github.com/envoyproxy/go-control-plane/... | grep 'v[2-9]' | xargs -n1 -I{} echo '\t_ \"{}\"' >> import_filters.gen.go"
//go:generate sh -c "echo ')' >> import_filters.gen.go"
