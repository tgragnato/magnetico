## [2.2.0](https://github.com/tgragnato/magnetico/compare/v2.1.0...v2.2.0) (2025-02-25)


### Features

* **persistence:** add database import functionality ([6f0cfb1](https://github.com/tgragnato/magnetico/commit/6f0cfb1141c36b27e5c9c85cbfb990a5e8a478d7))
* **persistence:** implement database export functionality ([c804674](https://github.com/tgragnato/magnetico/commit/c80467425edd3a8da0e9f228e624ab1c38b16a26))
* **persistence:** implement export method for database interfaces ([7a705a6](https://github.com/tgragnato/magnetico/commit/7a705a69e08b3c9e968a68f8807533715ddbfae7))
* **stats:** add pyroscope profiling support ([5d261e1](https://github.com/tgragnato/magnetico/commit/5d261e12df1e29445a03e9bc5478729efef2900d))
* **web:** add redirect for non-root paths and handle invalid methods ([3604006](https://github.com/tgragnato/magnetico/commit/360400601ab494a68accb06e455ceac2db060554))
* **web:** add timeout configuration for web interface and apis ([2255959](https://github.com/tgragnato/magnetico/commit/225595982cdf094762aabe99d809ca76c1104fe5))
* **web:** introduce a compression middleware ([a406108](https://github.com/tgragnato/magnetico/commit/a406108a9665858a68f05644838142cdde0996ad))
* **web:** set Content-Type header to text/html for HTML responses ([f6ee78a](https://github.com/tgragnato/magnetico/commit/f6ee78a8e2ef940b3f0e3bee6d4845fa096c4592))
* **web:** update router to specify HTTP methods and add robots.txt handler ([206757e](https://github.com/tgragnato/magnetico/commit/206757e9453e5d49c40d2bdaba214ec93e69e6b8))


### Bug Fixes

* **oci:** update dockerfile to use clang for building ([a368255](https://github.com/tgragnato/magnetico/commit/a3682556d16e1d784492672862e3acd61fa637f7))


### Performance Improvements

* **dht:** use swiss tables for nodes routing ([88b33ea](https://github.com/tgragnato/magnetico/commit/88b33ea8fcfcd5c7ac23101578031f3bb05d6c58))

## [2.1.0](https://github.com/tgragnato/magnetico/compare/v2.0.0...v2.1.0) (2025-01-04)


### Features

* add api endpoint to get total count of torrents based on query keyword ([bd5c138](https://github.com/tgragnato/magnetico/commit/bd5c138bba29c9e1b7430fc1fa82f4b2320c91e2))
* **web:** add new parameters with different semantics to the `torrentstotal` api ([fc60049](https://github.com/tgragnato/magnetico/commit/fc60049ad9f4ed417bdb2e186ddb22a270d98127))


### Bug Fixes

* **web:** correct property name for discoveredOn in torrent data ([b6bcc68](https://github.com/tgragnato/magnetico/commit/b6bcc6806af645edb9d7e62a9c7d4c20547e1961))
* **web:** don't use HTML entities when building URL (12c79539 fix-up) ([aff9ee5](https://github.com/tgragnato/magnetico/commit/aff9ee566fdb638252b954287109c180b71979c7))
* **web:** remove two debugging messages from the statistics page ([4d1097f](https://github.com/tgragnato/magnetico/commit/4d1097ffca156d04f87bf4b1069e96a07fd5b66d))


### Performance Improvements

* **persistence:** improve the count method for torrents ([0e995d4](https://github.com/tgragnato/magnetico/commit/0e995d4bd72b8aa361e9653ccf2f50472ad25441))

## [2.0.0](https://github.com/tgragnato/magnetico/compare/v1.62.0...v2.0.0) (2024-11-30)


### ⚠ BREAKING CHANGES

* review the logic related to the operational flags
* introduce the ability to use a configuration file
* allow user supplied ports for bootstrap nodes

### Features

* allow user supplied ports for bootstrap nodes ([774ed20](https://github.com/tgragnato/magnetico/commit/774ed206dd7c473a4004083b697f1e4c011b4f88))
* introduce the ability to use a configuration file ([6eb27c2](https://github.com/tgragnato/magnetico/commit/6eb27c2a17f8ff3db92b2e40cc413a4d779c63ea))
* **opflags:** add a flag for the deadline of leeches ([9709b18](https://github.com/tgragnato/magnetico/commit/9709b1863381fe0bf440cd4bb16eda99bf40db01))
* **persistence:** add rabbitmq amqp basic support ([24925ba](https://github.com/tgragnato/magnetico/commit/24925ba2f5896fffcef183067fa709733b998457))
* **persistence:** add support for the bitmagnet import api ([0a4579d](https://github.com/tgragnato/magnetico/commit/0a4579d496ce27f06e17c4d276d28bf8ddcf74f8))
* **persistence:** perform pattern matching without case sensitivity ([b27c8b1](https://github.com/tgragnato/magnetico/commit/b27c8b130aef60a134ef99d033d6e6222ff26dbf))
* **web:** add count parameter to rss feed ([8e7d2cf](https://github.com/tgragnato/magnetico/commit/8e7d2cfdb277aaff814df99e46b13859be72ac65))


### Bug Fixes

* **gci:** file is not `gci`-ed with --skip-generated -s standard -s default ([b746785](https://github.com/tgragnato/magnetico/commit/b746785568e1a40e348f78aff559ba87ca7bc840))
* **metadata:** avoid panic - return nil ([e3676f8](https://github.com/tgragnato/magnetico/commit/e3676f801bc6d37d94ae903a293e0c829a6fbf57))
* **metadata:** make V1Length avoid panic and return 0 ([586e548](https://github.com/tgragnato/magnetico/commit/586e54891525744eee4322ae96274e6352b42b2d))
* **metainfo:** return an error when the decoded length is wrong ([3f5f14d](https://github.com/tgragnato/magnetico/commit/3f5f14d1c626e9b74b8df4ff2a5fb8cf4be7a4f9))
* **opflags:** exit gracefully if invoked with the help flag ([ef502d6](https://github.com/tgragnato/magnetico/commit/ef502d6aea6127e2da0ddad6313c0404d6e96de5))
* **opflags:** restore the logic that sets both daemon and web to “on” if neither parameter is specified ([e130092](https://github.com/tgragnato/magnetico/commit/e130092009f6a9dc6672aa4431960ffb7eb75553))
* **persistence:** handle parameters passed to postgres with zero values ([dbd4032](https://github.com/tgragnato/magnetico/commit/dbd403245044e529de25f7fda6b03471fa05a29d))
* **web:** the rss feed should list the most common torrents ([4ebae27](https://github.com/tgragnato/magnetico/commit/4ebae270c766662694d8afb8334cc3b14ed6a0aa))


### Performance Improvements

* **persistence:** remove postgres parameters inherited from the sqlite implementation ([e8ab03b](https://github.com/tgragnato/magnetico/commit/e8ab03b302c3168fe220a433d789fd0e4692ae7e))


### Code Refactoring

* review the logic related to the operational flags ([0f02c84](https://github.com/tgragnato/magnetico/commit/0f02c8446b2e3199e7cdfe13fa9bb3690b3e3bad))

## [1.62.0](https://github.com/tgragnato/magnetico/compare/v1.61.2...v1.62.0) (2024-09-23)


### Features

* **dht:** implement NewSampleInfohashesResponse and onSampleInfohashesQuery ([6d69021](https://github.com/tgragnato/magnetico/commit/6d69021c26526a04855e7d35c0dfcc138172b02e))
* **dht:** introduce support for the ipv6 krpc protocol ([66fc7b0](https://github.com/tgragnato/magnetico/commit/66fc7b098eb3e09021df84dd0dc0e680f4554f73))
* **metainfo:** support for serializing v2 torrent file ([a8c2460](https://github.com/tgragnato/magnetico/commit/a8c2460727c2aeb01b16783342651d3606c10794))
* **stats:** add prometheus exporter for application metrics ([fc12657](https://github.com/tgragnato/magnetico/commit/fc126578bce118aba0ee1b19565984973a85ec28))


### Bug Fixes

* **bencode:** return empty value instead of panic ([52e9999](https://github.com/tgragnato/magnetico/commit/52e999914ad5115ad71d55e846796d8ec1bc22d3))
* **metadata:** avoid panic - return an error ([67946af](https://github.com/tgragnato/magnetico/commit/67946af3a01220aae560294e3454eda93d547ec9))
* **metadata:** report error when metadata size is zero ([e0fa28c](https://github.com/tgragnato/magnetico/commit/e0fa28c5f682b67a71c762f78e48d9f1d109e564))
* **metainfo:** empty Info marshalling test ([d1b3b3a](https://github.com/tgragnato/magnetico/commit/d1b3b3aa4dadecc84684446d370cebaab702d312))
* **metainfo:** stdlib integration ([535dc08](https://github.com/tgragnato/magnetico/commit/535dc08b6f81ccb070a2dfb056496d84e8a2cbbb))
* **persistence:** change signature to match the interface ([df4e1ee](https://github.com/tgragnato/magnetico/commit/df4e1eeac5053fdbc8e3920b2d033e0cbdd8a5c9))
* **persistence:** workaround for error with sqlite ≥ 3.34 ([f30db28](https://github.com/tgragnato/magnetico/commit/f30db28e0d11b76fd591c9868415a808fe1bdb38))


### Performance Improvements

* **metadata:** force mse - avoid throttling ([7c84ab4](https://github.com/tgragnato/magnetico/commit/7c84ab45eea6a91f77472f330f703dd727ed4968))

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

