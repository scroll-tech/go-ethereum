# CHANGELOGS

## 2022-05-04

Tag: None.

Current rev: a79e72f69701695185f2f71788d17998bdd5a5a8.

Based on https://github.com/ethereum/go-ethereum v1.10.13.

**Notable changes:**

### 1. Disable consensus and p2p service.

Related commits:

+ [a4eb31f4d2959a4ea97edf38aa13e4f87a81b1a1](https://github.com/scroll-tech/go-ethereum/commit/a4eb31f4d2959a4ea97edf38aa13e4f87a81b1a1) (PR [#8](https://github.com/scroll-tech/go-ethereum/pull/8))
+ [8b16b4cefae82f7d3f73a37df9e04d7d356a7a23](https://github.com/scroll-tech/go-ethereum/commit/8b16b4cefae82f7d3f73a37df9e04d7d356a7a23) (PR [#29](https://github.com/scroll-tech/go-ethereum/pull/29))

### 2. Add more detailed execution trace for zkevm-circuits proving.

Related commits:

+ [7745fd584018eb0ef63db1b17e27b968c9ae5dca](https://github.com/scroll-tech/go-ethereum/commit/7745fd584018eb0ef63db1b17e27b968c9ae5dca) (PR [#19](https://github.com/scroll-tech/go-ethereum/pull/19))
+ [69c291cf7ac43be7e10979457e4137af0c63ce0e](https://github.com/scroll-tech/go-ethereum/commit/69c291cf7ac43be7e10979457e4137af0c63ce0e) (PR [#20](https://github.com/scroll-tech/go-ethereum/pull/20))
+ [a5999cee905c16a24abb4580f43dca30ac9441d5](https://github.com/scroll-tech/go-ethereum/commit/a5999cee905c16a24abb4580f43dca30ac9441d5) (PR [#44](https://github.com/scroll-tech/go-ethereum/pull/44))
+ [13ea5c234b359e9eebe92b816ae6f02e876d6add](https://github.com/scroll-tech/go-ethereum/commit/13ea5c234b359e9eebe92b816ae6f02e876d6add) (PR [#46](https://github.com/scroll-tech/go-ethereum/pull/46))
+ [51281549254bfbfd3ad1dd87ffc9604c2b2e5a77](https://github.com/scroll-tech/go-ethereum/commit/51281549254bfbfd3ad1dd87ffc9604c2b2e5a77) (PR [#56](https://github.com/scroll-tech/go-ethereum/pull/56))
+ [8ccc10541dd49d52c70230cc0f8210d5da64248b](https://github.com/scroll-tech/go-ethereum/commit/8ccc10541dd49d52c70230cc0f8210d5da64248b) (PR [#58](https://github.com/scroll-tech/go-ethereum/pull/58))
+ [360115e61fb33aee60e02d7a907a2f0e79935ee2](https://github.com/scroll-tech/go-ethereum/commit/360115e61fb33aee60e02d7a907a2f0e79935ee2)
+ [9f1d8552e4d0abb30096ab3a75d9ab4616ac23b6](https://github.com/scroll-tech/go-ethereum/commit/9f1d8552e4d0abb30096ab3a75d9ab4616ac23b6) (PR [#66](https://github.com/scroll-tech/go-ethereum/pull/66))
+ [2329d324098f799e4d547d6ef14a77f52fc469cf](https://github.com/scroll-tech/go-ethereum/commit/2329d324098f799e4d547d6ef14a77f52fc469cf) (PR [#71](https://github.com/scroll-tech/go-ethereum/pull/71))
+ [d3bc8322dc503fa1b927a60b518f0b195641ffdf](https://github.com/scroll-tech/go-ethereum/commit/d3bc8322dc503fa1b927a60b518f0b195641ffdf) (PR [#102](https://github.com/scroll-tech/go-ethereum/pull/102))

(

And some fixes regarding encoding:

+ [06190d0642afe076b257d67ad40b6b85e0a6e087](https://github.com/scroll-tech/go-ethereum/commit/06190d0642afe076b257d67ad40b6b85e0a6e087) (PR [#75](https://github.com/scroll-tech/go-ethereum/pull/75))

Commits change:

+ [3b4875c0d5ce12909f7b29c219717bbe59024eee](https://github.com/scroll-tech/go-ethereum/commit/3b4875c0d5ce12909f7b29c219717bbe59024eee) (PR [#45](https://github.com/scroll-tech/go-ethereum/pull/45))
+ [f8532484a6887641b29a88fc8b0b7df06d4cf51b](https://github.com/scroll-tech/go-ethereum/commit/f8532484a6887641b29a88fc8b0b7df06d4cf51b) (PR [#48](https://github.com/scroll-tech/go-ethereum/pull/48))

Fields change:

+ [9fb413d70233e7bb5b627881ce867a3d1a11e90d](https://github.com/scroll-tech/go-ethereum/commit/9fb413d70233e7bb5b627881ce867a3d1a11e90d) (PR [#47](https://github.com/scroll-tech/go-ethereum/pull/47))
+ [e55d05d3ab9241caff76c0e576f5e6b6a6a5a743](https://github.com/scroll-tech/go-ethereum/commit/e55d05d3ab9241caff76c0e576f5e6b6a6a5a743) (PR [#63](https://github.com/scroll-tech/go-ethereum/pull/63))
+ [3d3c9d3edff7cc6c3445f4fc9cf072df7853ee7d](https://github.com/scroll-tech/go-ethereum/commit/3d3c9d3edff7cc6c3445f4fc9cf072df7853ee7d) (PR [#74](https://github.com/scroll-tech/go-ethereum/pull/74))
+ [571dcad4be512225bb1209f8008a8577eab29ded](https://github.com/scroll-tech/go-ethereum/commit/571dcad4be512225bb1209f8008a8577eab29ded) (PR [#98](https://github.com/scroll-tech/go-ethereum/pull/98))
+ [e15d0d35cba2aa6aab932df2691e7544e4ffda78](https://github.com/scroll-tech/go-ethereum/commit/e15d0d35cba2aa6aab932df2691e7544e4ffda78) (PR [#117](https://github.com/scroll-tech/go-ethereum/pull/117))

Bug fix:

+ [a96775b3ff6233da6c3662462bdb79eb13346004](https://github.com/scroll-tech/go-ethereum/commit/a96775b3ff6233da6c3662462bdb79eb13346004) (PR [#64](https://github.com/scroll-tech/go-ethereum/pull/64))
+ [fccb5bf6ec1564ee58838b93f900ed8e76fb48fd](https://github.com/scroll-tech/go-ethereum/commit/e55d05d3ab9241caff76c0e576f5e6b6a6a5a743) (PR [#67](https://github.com/scroll-tech/go-ethereum/pull/67))
+ [33fcd2bf6d4fa467bd8207bb1dc9c55bbed6be9b](https://github.com/scroll-tech/go-ethereum/commit/33fcd2bf6d4fa467bd8207bb1dc9c55bbed6be9b) (PR [#72](https://github.com/scroll-tech/go-ethereum/pull/72))
+ [f73142728206ddc4b89d3b3e9b5549933eba94fe](https://github.com/scroll-tech/go-ethereum/commit/f73142728206ddc4b89d3b3e9b5549933eba94fe) (PR [#119](https://github.com/scroll-tech/go-ethereum/pull/119))

)


### 3. Increase tps or reduce GC pressure

Related commits:

+ [09a31ccc66bcf676f71451bb1f3fde2e44849da3](https://github.com/scroll-tech/go-ethereum/commit/09a31ccc66bcf676f71451bb1f3fde2e44849da3) (PR [#43](https://github.com/scroll-tech/go-ethereum/pull/43))
+ [40e8f088b6adb8dcdc050759d978d19d18327a77](https://github.com/scroll-tech/go-ethereum/commit/40e8f088b6adb8dcdc050759d978d19d18327a77) (PR [#68](https://github.com/scroll-tech/go-ethereum/pull/68))
+ [a79e72f69701695185f2f71788d17998bdd5a5a8](https://github.com/scroll-tech/go-ethereum/commit/a79e72f69701695185f2f71788d17998bdd5a5a8) (PR [#83](https://github.com/scroll-tech/go-ethereum/pull/83))
+ [9199413d21c6c08f14ff968c472206e5ebff0518](https://github.com/scroll-tech/go-ethereum/commit/9199413d21c6c08f14ff968c472206e5ebff0518) (PR [#92](https://github.com/scroll-tech/go-ethereum/pull/92))
+ [9b99f2e17425fa16d1835cbfe47f9015321faae1](https://github.com/scroll-tech/go-ethereum/commit/9b99f2e17425fa16d1835cbfe47f9015321faae1) (PR [#104](https://github.com/scroll-tech/go-ethereum/pull/104))

### 4. Misc

4.1 Enable London fork rules from the beginning

Related commits:

+ [c180aa2e75d80dda90719b58690111b0d5b69f21](https://github.com/scroll-tech/go-ethereum/commit/c180aa2e75d80dda90719b58690111b0d5b69f21) (PR [#76](https://github.com/scroll-tech/go-ethereum/pull/76))

4.2 opcode operation

Related commits:

+ [21b65f4944667e574c29f37db6da7185b7dfa444](https://github.com/scroll-tech/go-ethereum/commit/21b65f4944667e574c29f37db6da7185b7dfa444) (PR [#118](https://github.com/scroll-tech/go-ethereum/pull/118))

4.3 The changes of module import

Related commits:

+ [de7ed6af40a9f90e19d2d691d40b07d5f356f81f](https://github.com/scroll-tech/go-ethereum/commit/de7ed6af40a9f90e19d2d691d40b07d5f356f81f) (PR [#15](https://github.com/scroll-tech/go-ethereum/pull/15))
+ [b4d60884d2a1259de0c3239d8cdaf33845937d93](https://github.com/scroll-tech/go-ethereum/commit/b4d60884d2a1259de0c3239d8cdaf33845937d93) (PR [#18](https://github.com/scroll-tech/go-ethereum/pull/18))
+ [59df2d7ebb0cb34688c1de4edacc577026b7fa4f](https://github.com/scroll-tech/go-ethereum/commit/59df2d7ebb0cb34688c1de4edacc577026b7fa4f) (PR [#22](https://github.com/scroll-tech/go-ethereum/pull/22))
+ [b6cc89fabe893bc9fdbd68ad3764432ae1bcd09a](https://github.com/scroll-tech/go-ethereum/commit/b6cc89fabe893bc9fdbd68ad3764432ae1bcd09a) (PR [#54](https://github.com/scroll-tech/go-ethereum/pull/54))
+ [9199413d21c6c08f14ff968c472206e5ebff0518](https://github.com/scroll-tech/go-ethereum/commit/9199413d21c6c08f14ff968c472206e5ebff0518) (PR [#92](https://github.com/scroll-tech/go-ethereum/pull/92))

4.4 The changes of ci、jenkins、docker、makefile and readme

Related commits:

+ [1ecbde748c8a77b7401872a911c24930c39dc350](https://github.com/scroll-tech/go-ethereum/commit/1ecbde748c8a77b7401872a911c24930c39dc350) (PR [#23](https://github.com/scroll-tech/go-ethereum/pull/23))
+ [2040f8c78170b0c0d95b95361ded7a5e4848ca2e](https://github.com/scroll-tech/go-ethereum/commit/2040f8c78170b0c0d95b95361ded7a5e4848ca2e) (PR [#32](https://github.com/scroll-tech/go-ethereum/pull/32))
+ [f07050dd0dea12f6da9ec30fb42e52f23bf4843a](https://github.com/scroll-tech/go-ethereum/commit/f07050dd0dea12f6da9ec30fb42e52f23bf4843a) (PR [#34](https://github.com/scroll-tech/go-ethereum/pull/34))
+ [35f6a91cd5d5bd2ecfc865f6c0c0b239727f55ee](https://github.com/scroll-tech/go-ethereum/commit/35f6a91cd5d5bd2ecfc865f6c0c0b239727f55ee) (PR [#111](https://github.com/scroll-tech/go-ethereum/pull/111))
+ [3410a56d866735f6a81eb8e5bae3976751ab0691](https://github.com/scroll-tech/go-ethereum/commit/) (PR [#121](https://github.com/scroll-tech/go-ethereum/pull/121))

4.5 The changes of genesis、config and readme

Related commits:

+ [ee15a2652fca6271e09c0f971a293bc1dcb51bde](https://github.com/scroll-tech/go-ethereum/commit/ee15a2652fca6271e09c0f971a293bc1dcb51bde) (PR [#1](https://github.com/scroll-tech/go-ethereum/pull/1))
+ [80940fa40dd2b65fd7f200c4bdfdc5ac3b57eabe](https://github.com/scroll-tech/go-ethereum/commit/80940fa40dd2b65fd7f200c4bdfdc5ac3b57eabe) (PR [#7](https://github.com/scroll-tech/go-ethereum/pull/7))
