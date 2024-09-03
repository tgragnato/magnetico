## [1.61.2](https://github.com/tgragnato/magnetico/compare/v1.61.1...v1.61.2) (2024-09-03)


### Bug Fixes

* **oci:** add some recommended labels ([ff520d6](https://github.com/tgragnato/magnetico/commit/ff520d661c0ea06188e2618780b44443530d627c))


### Performance Improvements

* **dht:** adaptive crawling rate ([32b6db8](https://github.com/tgragnato/magnetico/commit/32b6db833165d27673b18afd25976aaae3283c09))

## [1.61.1](https://github.com/tgragnato/magnetico/compare/v1.61.0...v1.61.1) (2024-09-01)


### Bug Fixes

* **dht:** this method cannot run in a goroutine because msg is shared ([f1cb43a](https://github.com/tgragnato/magnetico/commit/f1cb43a77b1c88bfddafc2f2b62354a37cc87c82))

## [1.61.0](https://github.com/tgragnato/magnetico/compare/v1.60.1...v1.61.0) (2024-09-01)


### Features

* **dht:** implement support for additional mainline methods ([db977b7](https://github.com/tgragnato/magnetico/commit/db977b7dcfb056e7f20604283f7a2a4b3c4ff381))
* flush metrics every minute ([de6001b](https://github.com/tgragnato/magnetico/commit/de6001b6fdaaa46f6dab8476813c7439bb3c59a6))
* **metadata:** add support for message stream encryption ([783dca8](https://github.com/tgragnato/magnetico/commit/783dca89fb4ff56ccb6b10f6a4e0e2df3d8acf03))
* **metadata:** add support for Multipath TCP ([e81f3b1](https://github.com/tgragnato/magnetico/commit/e81f3b1e4ccdfe0926de070b222a2629859ed7ba))
* **persistence:** implement a ZeroMQ interface ([a4654c2](https://github.com/tgragnato/magnetico/commit/a4654c2aeafc29e4aae2af3fb9696caebefce172))
* **persistence:** support both cgo and non-cgo builds ([96ee524](https://github.com/tgragnato/magnetico/commit/96ee524fe213076f02bfb42d35604e90c394cdef))
* **stats:** add a package for logging metrics ([c96820b](https://github.com/tgragnato/magnetico/commit/c96820b09179558ba2bfca34d3f879a7571a27a6))


### Bug Fixes

* **bencode:** impossible condition nil != nil ([c36c58b](https://github.com/tgragnato/magnetico/commit/c36c58b8af0ec0c1f836bc1f2817927b8f70154e))
* **dht:** correct slice reinitialization issue in CompactNodeInfo Marshaler ([a72de2c](https://github.com/tgragnato/magnetico/commit/a72de2c3c2a521f6365ea0645fb86dc2f9374bd8))
* **dht:** ignore nodes that have invalid ports ([819d2dd](https://github.com/tgragnato/magnetico/commit/819d2dd4e65b73d8252a22fa3992d38c36aa8f9b))
* **dht:** increase chan size but do not drop idx results ([8300868](https://github.com/tgragnato/magnetico/commit/830086829bfdcf24675ae25b93316f6ae36304df))
* **dht:** limit the number of nodes returned by dump ([1facb2b](https://github.com/tgragnato/magnetico/commit/1facb2b416505a0106b5a26e61a0f728e652fe0d))
* **infohash:** gracefully handle bad hex strings without panicking ([ab2e677](https://github.com/tgragnato/magnetico/commit/ab2e6772ac36de6e5b1c20758b4e59df773cff47))
* **metadata:** fallback on math/rand when crypto/rand fails ([f7c7dee](https://github.com/tgragnato/magnetico/commit/f7c7dee91c29301e28cfabc5ec72ee08de0535fe))
* **oci:** add zeromq dependencies in oci image build ([4ec62bb](https://github.com/tgragnato/magnetico/commit/4ec62bba9516543197c10b700486831ef02703ac))
* **oci:** avoid dev deps installation in the final container ([914ddd4](https://github.com/tgragnato/magnetico/commit/914ddd4f7b756507490cdfb99dede3f93b074db0))
* **oci:** the 'as' keyword should match the case of the 'from' keyword ([bf557ed](https://github.com/tgragnato/magnetico/commit/bf557ed8e6c159e62c035fabdcc4a6373ec453e4))
* **oom:** lower the default `indexer-max-neighbors` to 5000 peers ([132bf1b](https://github.com/tgragnato/magnetico/commit/132bf1bbdb3b5fea22f5b1e923596263ad731237))
* **persistence:** replace zeromq deps with the gopkg.in counterpart ([a75f859](https://github.com/tgragnato/magnetico/commit/a75f859615c8014cbd99b5cf96b5e2146c524ab4))
* **persistence:** return an invalid value - avoid panics ([7e43e10](https://github.com/tgragnato/magnetico/commit/7e43e105f2dfec60429efa9625a0f6f4a14c22df))
* **persistence:** rows might be nil if err != nil ([8f09f8e](https://github.com/tgragnato/magnetico/commit/8f09f8eefe377af1ac68381d805bc73625da7251))
* **persistence:** rows might be nil if err != nil ([c61d863](https://github.com/tgragnato/magnetico/commit/c61d86319e0eaaaeb17455c706394efa4d9df7e6))
* remove go:build statement ([2c09c90](https://github.com/tgragnato/magnetico/commit/2c09c90ff20e7a768d068a4063c6f50f9f9cd177))


### Performance Improvements

* **dht:** `rt.nodes` - avoid excessive allocations ([3ced3c6](https://github.com/tgragnato/magnetico/commit/3ced3c66d4a8c5392c23982614f9120ed3f7de51))
* **dht:** add slice of neighbors in the routing table ([f115de1](https://github.com/tgragnato/magnetico/commit/f115de1a7ea4973fd7a7fde73bf30213113b12ce))
* **dht:** discard old entries in the routing table if it gets too big ([1339624](https://github.com/tgragnato/magnetico/commit/13396242fb1972a928b11d44c792bac659d26ac7))
* **dht:** on sample_infohash query received send a find_node query ([dc0e761](https://github.com/tgragnato/magnetico/commit/dc0e76124284bf63a07bc6d14257534aadcc19b4))
* **dht:** prioritize `get_peers` and `find_node` queries ([565801a](https://github.com/tgragnato/magnetico/commit/565801a86f9767a06dfd6e25437a5b9779512947))

## [1.60.1](https://github.com/tgragnato/magnetico/compare/v1.60.0...v1.60.1) (2024-08-14)


### Bug Fixes

* **deps:** drop a benchmark to unload a bunch of dependencies ([61c3239](https://github.com/tgragnato/magnetico/commit/61c3239ff2c7a3714b769391408b4c68643c44b8))
* **web:** drop the assets related to the readme endpoint ([a1884eb](https://github.com/tgragnato/magnetico/commit/a1884eb8c15dcbf83c213164e107e122bdad8509))

## [1.60.0](https://github.com/tgragnato/magnetico/compare/v1.59.0...v1.60.0) (2024-08-13)


### Features

* add arm64 to multi-platform container builds ([4a09971](https://github.com/tgragnato/magnetico/commit/4a09971b9f9ecd26c7dc3f40c5290b0f6a6138c3))
* **ci:** publish multi-platform executables in releases ([89f7f02](https://github.com/tgragnato/magnetico/commit/89f7f020b6ccb20f80b54127d41c5004d542e5b1))


### Bug Fixes

* **ci:** enable cache support ([a8d2a49](https://github.com/tgragnato/magnetico/commit/a8d2a49764e603d3f3cda07303b346e6fd360254))
* **ci:** replace actions/delete-package-versions with dataaxiom/ghcr-cleaner-action ([8838565](https://github.com/tgragnato/magnetico/commit/8838565c4bd197c0466b39a1ae38c3258a02ffde))

## [1.59.0](https://github.com/tgragnato/magnetico/compare/v1.58.3...v1.59.0) (2024-08-13)


### Features

* add validation for filter-nodes-cidrs during parsing ([50e7717](https://github.com/tgragnato/magnetico/commit/50e77170a178e0c588c16cc3f19368896e06100b))
* **dht:** crawl only specified ip ranges ([acb1913](https://github.com/tgragnato/magnetico/commit/acb19137b5776343b8fa087615ee95ab4436eb70))


### Bug Fixes

* invalid Go toolchain version ([6a92690](https://github.com/tgragnato/magnetico/commit/6a92690cb4f79355a679327ecbad4b55d9b7e2e0))

