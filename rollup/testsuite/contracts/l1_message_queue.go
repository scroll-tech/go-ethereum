// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// L1MessageQueueMetaData contains all meta data concerning the L1MessageQueue contract.
var L1MessageQueueMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_messenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_scrollChain\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_enforcedTxGateway\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"appendCrossDomainMessage\",\"inputs\":[{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"appendEnforcedTransaction\",\"inputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"calculateIntrinsicGasFee\",\"inputs\":[{\"name\":\"_calldata\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"computeTransactionHash\",\"inputs\":[{\"name\":\"_sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"dropCrossDomainMessage\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"enforcedTxGateway\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"estimateCrossDomainMessageFee\",\"inputs\":[{\"name\":\"_gasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"finalizePoppedCrossDomainMessage\",\"inputs\":[{\"name\":\"_newFinalizedQueueIndexPlusOne\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"gasOracle\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCrossDomainMessage\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_messenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_scrollChain\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_enforcedTxGateway\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_gasOracle\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_maxGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isMessageDropped\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isMessageSkipped\",\"inputs\":[{\"name\":\"_queueIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"maxGasLimit\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messageQueue\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"messenger\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextCrossDomainMessageIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"nextUnfinalizedQueueIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"pendingQueueIndex\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"popCrossDomainMessage\",\"inputs\":[{\"name\":\"_startIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_count\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_skippedBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resetPoppedCrossDomainMessage\",\"inputs\":[{\"name\":\"_startIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"scrollChain\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateGasOracle\",\"inputs\":[{\"name\":\"_newGasOracle\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateMaxGasLimit\",\"inputs\":[{\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"DequeueTransaction\",\"inputs\":[{\"name\":\"startIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"count\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"skippedBitmap\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DropTransaction\",\"inputs\":[{\"name\":\"index\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FinalizedDequeuedTransaction\",\"inputs\":[{\"name\":\"finalizedIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"QueueTransaction\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"target\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"queueIndex\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"gasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ResetDequeuedTransaction\",\"inputs\":[{\"name\":\"startIndex\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UpdateGasOracle\",\"inputs\":[{\"name\":\"_oldGasOracle\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"_newGasOracle\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UpdateMaxGasLimit\",\"inputs\":[{\"name\":\"_oldMaxGasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"_newMaxGasLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ErrorZeroAddress\",\"inputs\":[]}]",
	Bin: "0x60e060405234801562000010575f80fd5b50604051620019c3380380620019c38339810160408190526200003391620000bd565b6001600160a01b03831615806200005157506001600160a01b038216155b806200006457506001600160a01b038116155b15620000835760405163a7f9319d60e01b815260040160405180910390fd5b6001600160a01b0392831660805290821660a0521660c05262000104565b80516001600160a01b0381168114620000b8575f80fd5b919050565b5f805f60608486031215620000d0575f80fd5b620000db84620000a1565b9250620000eb60208501620000a1565b9150620000fb60408501620000a1565b90509250925092565b60805160a05160c05161186a620001595f395f81816102470152610d9c01525f81816102fa0152818161040b01528181610594015261097101525f81816101e501528181610b3d0152610cf9015261186a5ff3fe608060405234801561000f575f80fd5b50600436106101a1575f3560e01c80637d82191a116100f3578063bdc6f0a011610093578063e172d3a11161006e578063e172d3a1146103a8578063f2fde38b146103bb578063f7013ef6146103ce578063fd0ad31e146103e1575f80fd5b8063bdc6f0a01461036f578063d5ad4a9714610382578063d7704bae14610395575f80fd5b806391652461116100ce578063916524611461032d5780639b15978214610340578063a85006ca14610353578063ae453cd51461035c575f80fd5b80637d82191a146102e2578063897630dd146102f55780638da5cb5b1461031c575f80fd5b806355f613ce1161015e5780635e45da23116101395780635e45da23146102ab57806370cee67f146102b4578063715018a6146102c75780637a6e9333146102cf575f80fd5b806355f613ce146102725780635ad9945a146102855780635d62a8dd14610298575f80fd5b806329aa604b146101a557806338050fd4146101cb5780633cb747bf146101e05780633e6dada11461021f5780633e83496c14610242578063416bdfa114610269575b5f80fd5b6101b86101b336600461143e565b6103e9565b6040519081526020015b60405180910390f35b6101de6101d936600461143e565b610408565b005b6102077f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b0390911681526020016101c2565b61023261022d36600461143e565b610549565b60405190151581526020016101c2565b6102077f000000000000000000000000000000000000000000000000000000000000000081565b6101b8606e5481565b6101de610280366004611455565b610591565b6101b86102933660046114d9565b610712565b606854610207906001600160a01b031681565b6101b8606b5481565b6101de6102c2366004611555565b610902565b6101de61095b565b6101de6102dd36600461143e565b61096e565b6102326102f036600461143e565b610b07565b6102077f000000000000000000000000000000000000000000000000000000000000000081565b6033546001600160a01b0316610207565b6101de61033b36600461143e565b610b3a565b6101de61034e36600461156e565b610cf6565b6101b8606a5481565b6101b861036a36600461143e565b610d75565b6101de61037d3660046115c4565b610d99565b6101de61039036600461143e565b610e84565b6101b86103a336600461143e565b610eca565b6101b86103b6366004611637565b610f53565b6101de6103c9366004611555565b610fe4565b6101de6103dc366004611676565b61105a565b6069546101b8565b606981815481106103f8575f80fd5b5f91825260209091200154905081565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316146104595760405162461bcd60e51b8152600401610450906116ce565b60405180910390fd5b606e54808203610467575050565b8082116104b65760405162461bcd60e51b815260206004820152601960248201527f66696e616c697a656420696e64657820746f6f20736d616c6c000000000000006044820152606401610450565b606a548211156105085760405162461bcd60e51b815260206004820152601960248201527f66696e616c697a656420696e64657820746f6f206c61726765000000000000006044820152606401610450565b606e8290556040515f19830181527fbbbf2de085aff601d965315326f9908eb5ebbb3d1b307e7e5ec42384e3320a10906020015b60405180910390a1505b50565b600881901c5f908152606d6020526040812054600160ff84161b161515801561058b5750600882901c5f908152606c6020526040902054600160ff84161b1615155b92915050565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316146105d95760405162461bcd60e51b8152600401610450906116ce565b6101008211156106235760405162461bcd60e51b8152602060048201526015602482015274706f7020746f6f206d616e79206d6573736167657360581b6044820152606401610450565b82606a541461066b5760405162461bcd60e51b81526020600482015260146024820152730e6e8c2e4e840d2dcc8caf040dad2e6dac2e8c6d60631b6044820152606401610450565b600883901c5f818152606d6020526040902080546001851b5f190193841660ff871681811b9092179092559092919061010081860111156106c357600182015f908152606d6020526040902061010082900385901c90555b505050818301606a5560408051848152602081018490529081018290527fc77f792f838ae38399ac31acc3348389aeb110ce7bedf3cfdbdd5e66792679709060600160405180910390a1505050565b5f607e816107bc565b5f8161072957506001919050565b5b811561073f5760089190911c9060010161072a565b919050565b80608083106001811461077c5761075a8461071b565b60808101835360018301925084816020036008021b835280830192505061079d565b848415166001811461079057848353610795565b608083535b506001820191505b509392505050565b806094815360609290921b60018301525060150190565b600560405101806107cf60018c83610744565b90506107dd60018983610744565b90506107e989826107a5565b90506107f760018b83610744565b9050600186146001811461085f5760388710600181146108445761081a8861071b565b8060b701845360018401935088816020036008021b84528084019350508789843791870191610859565b87608001835360018301925087898437918701915b50610870565b61086d5f89355f1a84610744565b91505b5061087b8c826107a5565b90508181035f8060388310600181146108ae576108978461071b565b60f78101600882021b8517935060010191506108b9565b8360c0019250600191505b5086816008021b821791506001810190508060080292508451831c8284610100031b17915080850394505080845250508181038220925050508092505050979650505050505050565b61090a6111b8565b606880546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e905f90a35050565b6109636111b8565b61096c5f611212565b565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316146109b65760405162461bcd60e51b8152600401610450906116ce565b606a548082036109c4575050565b606e54821015610a165760405162461bcd60e51b815260206004820152601860248201527f72657365742066696e616c697a6564206d6573736167657300000000000000006044820152606401610450565b808210610a5e5760405162461bcd60e51b815260206004820152601660248201527572657365742070656e64696e67206d6573736167657360501b6044820152606401610450565b600882901c5f818152606d602052604090208054600160ff861690811b5f190190911690915583830391906101008190035b83811015610ace576001929092015f818152606d60205260409020549092908015610ac4575f848152606d60205260408120555b5061010001610a90565b505050606a839055506040518281527fc079f1a662217305bfe03e0a85f03944a2ac422f5ee5431c98b9ef7d3c6226c99060200161053c565b5f606a548210610b1857505f919050565b600882901c5f908152606d6020526040902054600160ff84161b16151561058b565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031614610b825760405162461bcd60e51b815260040161045090611703565b606e548110610bd35760405162461bcd60e51b815260206004820152601b60248201527f63616e6e6f742064726f702070656e64696e67206d65737361676500000000006044820152606401610450565b600881901c5f908152606d6020526040902054600160ff83161b16610c3a5760405162461bcd60e51b815260206004820152601860248201527f64726f70206e6f6e2d736b6970706564206d65737361676500000000000000006044820152606401610450565b600881901c5f908152606c6020526040902054600160ff83161b1615610ca25760405162461bcd60e51b815260206004820152601760248201527f6d65737361676520616c72656164792064726f707065640000000000000000006044820152606401610450565b600881901c5f908152606c602052604090208054600160ff84161b1790556040518181527f43a375005206d20a83abc71722cba68c24434a8dc1f583775be7c3fde0396cbf9060200160405180910390a150565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031614610d3e5760405162461bcd60e51b815260040161045090611703565b610d49838383611263565b3373111100000000000000000000000000000000111101610d6e81865f878787611342565b5050505050565b5f60698281548110610d8957610d89611749565b905f5260205f2001549050919050565b337f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031614610e205760405162461bcd60e51b815260206004820152602660248201527f4f6e6c792063616c6c61626c652062792074686520456e666f7263656454784760448201526561746577617960d01b6064820152608401610450565b6001600160a01b0386163b15610e635760405162461bcd60e51b81526020600482015260086024820152676f6e6c7920454f4160c01b6044820152606401610450565b610e6e838383611263565b610e7c868686868686611342565b505050505050565b610e8c6111b8565b606b80549082905560408051828152602081018490527fa030881e03ff723954dd0d35500564afab9603555d09d4456a32436f2b2373c5910161053c565b6068545f906001600160a01b031680610ee557505f92915050565b604051636bb825d760e11b8152600481018490526001600160a01b0382169063d7704bae90602401602060405180830381865afa158015610f28573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610f4c919061175d565b9392505050565b6068545f906001600160a01b031680610f6f575f91505061058b565b60405163e172d3a160e01b81526001600160a01b0382169063e172d3a190610f9d908790879060040161179c565b602060405180830381865afa158015610fb8573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610fdc919061175d565b949350505050565b610fec6111b8565b6001600160a01b0381166110515760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b6064820152608401610450565b61054681611212565b5f54610100900460ff161580801561107857505f54600160ff909116105b806110915750303b15801561109157505f5460ff166001145b6110f45760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401610450565b5f805460ff191660011790558015611115575f805461ff0019166101001790555b61111d6113e6565b606880546001600160a01b038086166001600160a01b031992831617909255606b849055606580548984169083161790556066805488841690831617905560678054928716929091169190911790558015610e7c575f805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a1505050505050565b6033546001600160a01b0316331461096c5760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610450565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0905f90a35050565b606b548311156112c35760405162461bcd60e51b815260206004820152602560248201527f476173206c696d6974206d757374206e6f7420657863656564206d6178476173604482015264131a5b5a5d60da1b6064820152608401610450565b5f6112ce8383610f53565b90508084101561133c5760405162461bcd60e51b815260206004820152603360248201527f496e73756666696369656e7420676173206c696d69742c206d7573742062652060448201527261626f766520696e7472696e7369632067617360681b6064820152608401610450565b50505050565b6069545f6113558883888a898989610712565b606980546001810182555f919091527f7fb4302e8e91f9110a6554c2c0a24601252c2a42c2220ca988efcfe399914308018190556040519091506001600160a01b0380891691908a16907f69cfcb8e6d4192b8aba9902243912587f37e550d75c1fa801491fce26717f37e906113d4908a9087908b908b908b906117af565b60405180910390a35050505050505050565b5f54610100900460ff1661140c5760405162461bcd60e51b8152600401610450906117e9565b61096c5f54610100900460ff166114355760405162461bcd60e51b8152600401610450906117e9565b61096c33611212565b5f6020828403121561144e575f80fd5b5035919050565b5f805f60608486031215611467575f80fd5b505081359360208301359350604090920135919050565b80356001600160a01b038116811461073f575f80fd5b5f8083601f8401126114a4575f80fd5b50813567ffffffffffffffff8111156114bb575f80fd5b6020830191508360208285010111156114d2575f80fd5b9250929050565b5f805f805f805f60c0888a0312156114ef575f80fd5b6114f88861147e565b965060208801359550604088013594506115146060890161147e565b93506080880135925060a088013567ffffffffffffffff811115611536575f80fd5b6115428a828b01611494565b989b979a50959850939692959293505050565b5f60208284031215611565575f80fd5b610f4c8261147e565b5f805f8060608587031215611581575f80fd5b61158a8561147e565b935060208501359250604085013567ffffffffffffffff8111156115ac575f80fd5b6115b887828801611494565b95989497509550505050565b5f805f805f8060a087890312156115d9575f80fd5b6115e28761147e565b95506115f06020880161147e565b94506040870135935060608701359250608087013567ffffffffffffffff811115611619575f80fd5b61162589828a01611494565b979a9699509497509295939492505050565b5f8060208385031215611648575f80fd5b823567ffffffffffffffff81111561165e575f80fd5b61166a85828601611494565b90969095509350505050565b5f805f805f60a0868803121561168a575f80fd5b6116938661147e565b94506116a16020870161147e565b93506116af6040870161147e565b92506116bd6060870161147e565b949793965091946080013592915050565b6020808252818101527f4f6e6c792063616c6c61626c6520627920746865205363726f6c6c436861696e604082015260600190565b60208082526026908201527f4f6e6c792063616c6c61626c6520627920746865204c315363726f6c6c4d657360408201526539b2b733b2b960d11b606082015260800190565b634e487b7160e01b5f52603260045260245ffd5b5f6020828403121561176d575f80fd5b5051919050565b81835281816020850137505f828201602090810191909152601f909101601f19169091010190565b602081525f610fdc602083018486611774565b85815267ffffffffffffffff85166020820152836040820152608060608201525f6117de608083018486611774565b979650505050505050565b6020808252602b908201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960408201526a6e697469616c697a696e6760a81b60608201526080019056fea264697066735822122001154f5318974e5780e3c9198dd07af49f60802435bb5485b2cf5a4d8533c59e64736f6c63430008180033",
}

// L1MessageQueueABI is the input ABI used to generate the binding from.
// Deprecated: Use L1MessageQueueMetaData.ABI instead.
var L1MessageQueueABI = L1MessageQueueMetaData.ABI

// L1MessageQueueBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1MessageQueueMetaData.Bin instead.
var L1MessageQueueBin = L1MessageQueueMetaData.Bin

// DeployL1MessageQueue deploys a new Ethereum contract, binding an instance of L1MessageQueue to it.
func DeployL1MessageQueue(auth *bind.TransactOpts, backend bind.ContractBackend, _messenger common.Address, _scrollChain common.Address, _enforcedTxGateway common.Address) (common.Address, *types.Transaction, *L1MessageQueue, error) {
	parsed, err := L1MessageQueueMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1MessageQueueBin), backend, _messenger, _scrollChain, _enforcedTxGateway)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1MessageQueue{L1MessageQueueCaller: L1MessageQueueCaller{contract: contract}, L1MessageQueueTransactor: L1MessageQueueTransactor{contract: contract}, L1MessageQueueFilterer: L1MessageQueueFilterer{contract: contract}}, nil
}

// L1MessageQueue is an auto generated Go binding around an Ethereum contract.
type L1MessageQueue struct {
	L1MessageQueueCaller     // Read-only binding to the contract
	L1MessageQueueTransactor // Write-only binding to the contract
	L1MessageQueueFilterer   // Log filterer for contract events
}

// L1MessageQueueCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1MessageQueueCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1MessageQueueTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1MessageQueueTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1MessageQueueFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1MessageQueueFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1MessageQueueSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1MessageQueueSession struct {
	Contract     *L1MessageQueue   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L1MessageQueueCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1MessageQueueCallerSession struct {
	Contract *L1MessageQueueCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L1MessageQueueTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1MessageQueueTransactorSession struct {
	Contract     *L1MessageQueueTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L1MessageQueueRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1MessageQueueRaw struct {
	Contract *L1MessageQueue // Generic contract binding to access the raw methods on
}

// L1MessageQueueCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1MessageQueueCallerRaw struct {
	Contract *L1MessageQueueCaller // Generic read-only contract binding to access the raw methods on
}

// L1MessageQueueTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1MessageQueueTransactorRaw struct {
	Contract *L1MessageQueueTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1MessageQueue creates a new instance of L1MessageQueue, bound to a specific deployed contract.
func NewL1MessageQueue(address common.Address, backend bind.ContractBackend) (*L1MessageQueue, error) {
	contract, err := bindL1MessageQueue(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueue{L1MessageQueueCaller: L1MessageQueueCaller{contract: contract}, L1MessageQueueTransactor: L1MessageQueueTransactor{contract: contract}, L1MessageQueueFilterer: L1MessageQueueFilterer{contract: contract}}, nil
}

// NewL1MessageQueueCaller creates a new read-only instance of L1MessageQueue, bound to a specific deployed contract.
func NewL1MessageQueueCaller(address common.Address, caller bind.ContractCaller) (*L1MessageQueueCaller, error) {
	contract, err := bindL1MessageQueue(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueCaller{contract: contract}, nil
}

// NewL1MessageQueueTransactor creates a new write-only instance of L1MessageQueue, bound to a specific deployed contract.
func NewL1MessageQueueTransactor(address common.Address, transactor bind.ContractTransactor) (*L1MessageQueueTransactor, error) {
	contract, err := bindL1MessageQueue(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueTransactor{contract: contract}, nil
}

// NewL1MessageQueueFilterer creates a new log filterer instance of L1MessageQueue, bound to a specific deployed contract.
func NewL1MessageQueueFilterer(address common.Address, filterer bind.ContractFilterer) (*L1MessageQueueFilterer, error) {
	contract, err := bindL1MessageQueue(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueFilterer{contract: contract}, nil
}

// bindL1MessageQueue binds a generic wrapper to an already deployed contract.
func bindL1MessageQueue(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := L1MessageQueueMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1MessageQueue *L1MessageQueueRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1MessageQueue.Contract.L1MessageQueueCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1MessageQueue *L1MessageQueueRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.L1MessageQueueTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1MessageQueue *L1MessageQueueRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.L1MessageQueueTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1MessageQueue *L1MessageQueueCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1MessageQueue.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1MessageQueue *L1MessageQueueTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1MessageQueue *L1MessageQueueTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.contract.Transact(opts, method, params...)
}

// CalculateIntrinsicGasFee is a free data retrieval call binding the contract method 0xe172d3a1.
//
// Solidity: function calculateIntrinsicGasFee(bytes _calldata) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) CalculateIntrinsicGasFee(opts *bind.CallOpts, _calldata []byte) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "calculateIntrinsicGasFee", _calldata)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CalculateIntrinsicGasFee is a free data retrieval call binding the contract method 0xe172d3a1.
//
// Solidity: function calculateIntrinsicGasFee(bytes _calldata) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) CalculateIntrinsicGasFee(_calldata []byte) (*big.Int, error) {
	return _L1MessageQueue.Contract.CalculateIntrinsicGasFee(&_L1MessageQueue.CallOpts, _calldata)
}

// CalculateIntrinsicGasFee is a free data retrieval call binding the contract method 0xe172d3a1.
//
// Solidity: function calculateIntrinsicGasFee(bytes _calldata) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) CalculateIntrinsicGasFee(_calldata []byte) (*big.Int, error) {
	return _L1MessageQueue.Contract.CalculateIntrinsicGasFee(&_L1MessageQueue.CallOpts, _calldata)
}

// ComputeTransactionHash is a free data retrieval call binding the contract method 0x5ad9945a.
//
// Solidity: function computeTransactionHash(address _sender, uint256 _queueIndex, uint256 _value, address _target, uint256 _gasLimit, bytes _data) pure returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCaller) ComputeTransactionHash(opts *bind.CallOpts, _sender common.Address, _queueIndex *big.Int, _value *big.Int, _target common.Address, _gasLimit *big.Int, _data []byte) ([32]byte, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "computeTransactionHash", _sender, _queueIndex, _value, _target, _gasLimit, _data)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ComputeTransactionHash is a free data retrieval call binding the contract method 0x5ad9945a.
//
// Solidity: function computeTransactionHash(address _sender, uint256 _queueIndex, uint256 _value, address _target, uint256 _gasLimit, bytes _data) pure returns(bytes32)
func (_L1MessageQueue *L1MessageQueueSession) ComputeTransactionHash(_sender common.Address, _queueIndex *big.Int, _value *big.Int, _target common.Address, _gasLimit *big.Int, _data []byte) ([32]byte, error) {
	return _L1MessageQueue.Contract.ComputeTransactionHash(&_L1MessageQueue.CallOpts, _sender, _queueIndex, _value, _target, _gasLimit, _data)
}

// ComputeTransactionHash is a free data retrieval call binding the contract method 0x5ad9945a.
//
// Solidity: function computeTransactionHash(address _sender, uint256 _queueIndex, uint256 _value, address _target, uint256 _gasLimit, bytes _data) pure returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCallerSession) ComputeTransactionHash(_sender common.Address, _queueIndex *big.Int, _value *big.Int, _target common.Address, _gasLimit *big.Int, _data []byte) ([32]byte, error) {
	return _L1MessageQueue.Contract.ComputeTransactionHash(&_L1MessageQueue.CallOpts, _sender, _queueIndex, _value, _target, _gasLimit, _data)
}

// EnforcedTxGateway is a free data retrieval call binding the contract method 0x3e83496c.
//
// Solidity: function enforcedTxGateway() view returns(address)
func (_L1MessageQueue *L1MessageQueueCaller) EnforcedTxGateway(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "enforcedTxGateway")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// EnforcedTxGateway is a free data retrieval call binding the contract method 0x3e83496c.
//
// Solidity: function enforcedTxGateway() view returns(address)
func (_L1MessageQueue *L1MessageQueueSession) EnforcedTxGateway() (common.Address, error) {
	return _L1MessageQueue.Contract.EnforcedTxGateway(&_L1MessageQueue.CallOpts)
}

// EnforcedTxGateway is a free data retrieval call binding the contract method 0x3e83496c.
//
// Solidity: function enforcedTxGateway() view returns(address)
func (_L1MessageQueue *L1MessageQueueCallerSession) EnforcedTxGateway() (common.Address, error) {
	return _L1MessageQueue.Contract.EnforcedTxGateway(&_L1MessageQueue.CallOpts)
}

// EstimateCrossDomainMessageFee is a free data retrieval call binding the contract method 0xd7704bae.
//
// Solidity: function estimateCrossDomainMessageFee(uint256 _gasLimit) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) EstimateCrossDomainMessageFee(opts *bind.CallOpts, _gasLimit *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "estimateCrossDomainMessageFee", _gasLimit)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// EstimateCrossDomainMessageFee is a free data retrieval call binding the contract method 0xd7704bae.
//
// Solidity: function estimateCrossDomainMessageFee(uint256 _gasLimit) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) EstimateCrossDomainMessageFee(_gasLimit *big.Int) (*big.Int, error) {
	return _L1MessageQueue.Contract.EstimateCrossDomainMessageFee(&_L1MessageQueue.CallOpts, _gasLimit)
}

// EstimateCrossDomainMessageFee is a free data retrieval call binding the contract method 0xd7704bae.
//
// Solidity: function estimateCrossDomainMessageFee(uint256 _gasLimit) view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) EstimateCrossDomainMessageFee(_gasLimit *big.Int) (*big.Int, error) {
	return _L1MessageQueue.Contract.EstimateCrossDomainMessageFee(&_L1MessageQueue.CallOpts, _gasLimit)
}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1MessageQueue *L1MessageQueueCaller) GasOracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "gasOracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1MessageQueue *L1MessageQueueSession) GasOracle() (common.Address, error) {
	return _L1MessageQueue.Contract.GasOracle(&_L1MessageQueue.CallOpts)
}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1MessageQueue *L1MessageQueueCallerSession) GasOracle() (common.Address, error) {
	return _L1MessageQueue.Contract.GasOracle(&_L1MessageQueue.CallOpts)
}

// GetCrossDomainMessage is a free data retrieval call binding the contract method 0xae453cd5.
//
// Solidity: function getCrossDomainMessage(uint256 _queueIndex) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCaller) GetCrossDomainMessage(opts *bind.CallOpts, _queueIndex *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "getCrossDomainMessage", _queueIndex)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetCrossDomainMessage is a free data retrieval call binding the contract method 0xae453cd5.
//
// Solidity: function getCrossDomainMessage(uint256 _queueIndex) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueSession) GetCrossDomainMessage(_queueIndex *big.Int) ([32]byte, error) {
	return _L1MessageQueue.Contract.GetCrossDomainMessage(&_L1MessageQueue.CallOpts, _queueIndex)
}

// GetCrossDomainMessage is a free data retrieval call binding the contract method 0xae453cd5.
//
// Solidity: function getCrossDomainMessage(uint256 _queueIndex) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCallerSession) GetCrossDomainMessage(_queueIndex *big.Int) ([32]byte, error) {
	return _L1MessageQueue.Contract.GetCrossDomainMessage(&_L1MessageQueue.CallOpts, _queueIndex)
}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3e6dada1.
//
// Solidity: function isMessageDropped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueCaller) IsMessageDropped(opts *bind.CallOpts, _queueIndex *big.Int) (bool, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "isMessageDropped", _queueIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3e6dada1.
//
// Solidity: function isMessageDropped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueSession) IsMessageDropped(_queueIndex *big.Int) (bool, error) {
	return _L1MessageQueue.Contract.IsMessageDropped(&_L1MessageQueue.CallOpts, _queueIndex)
}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3e6dada1.
//
// Solidity: function isMessageDropped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueCallerSession) IsMessageDropped(_queueIndex *big.Int) (bool, error) {
	return _L1MessageQueue.Contract.IsMessageDropped(&_L1MessageQueue.CallOpts, _queueIndex)
}

// IsMessageSkipped is a free data retrieval call binding the contract method 0x7d82191a.
//
// Solidity: function isMessageSkipped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueCaller) IsMessageSkipped(opts *bind.CallOpts, _queueIndex *big.Int) (bool, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "isMessageSkipped", _queueIndex)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageSkipped is a free data retrieval call binding the contract method 0x7d82191a.
//
// Solidity: function isMessageSkipped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueSession) IsMessageSkipped(_queueIndex *big.Int) (bool, error) {
	return _L1MessageQueue.Contract.IsMessageSkipped(&_L1MessageQueue.CallOpts, _queueIndex)
}

// IsMessageSkipped is a free data retrieval call binding the contract method 0x7d82191a.
//
// Solidity: function isMessageSkipped(uint256 _queueIndex) view returns(bool)
func (_L1MessageQueue *L1MessageQueueCallerSession) IsMessageSkipped(_queueIndex *big.Int) (bool, error) {
	return _L1MessageQueue.Contract.IsMessageSkipped(&_L1MessageQueue.CallOpts, _queueIndex)
}

// MaxGasLimit is a free data retrieval call binding the contract method 0x5e45da23.
//
// Solidity: function maxGasLimit() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) MaxGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "maxGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxGasLimit is a free data retrieval call binding the contract method 0x5e45da23.
//
// Solidity: function maxGasLimit() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) MaxGasLimit() (*big.Int, error) {
	return _L1MessageQueue.Contract.MaxGasLimit(&_L1MessageQueue.CallOpts)
}

// MaxGasLimit is a free data retrieval call binding the contract method 0x5e45da23.
//
// Solidity: function maxGasLimit() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) MaxGasLimit() (*big.Int, error) {
	return _L1MessageQueue.Contract.MaxGasLimit(&_L1MessageQueue.CallOpts)
}

// MessageQueue is a free data retrieval call binding the contract method 0x29aa604b.
//
// Solidity: function messageQueue(uint256 ) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCaller) MessageQueue(opts *bind.CallOpts, arg0 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "messageQueue", arg0)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// MessageQueue is a free data retrieval call binding the contract method 0x29aa604b.
//
// Solidity: function messageQueue(uint256 ) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueSession) MessageQueue(arg0 *big.Int) ([32]byte, error) {
	return _L1MessageQueue.Contract.MessageQueue(&_L1MessageQueue.CallOpts, arg0)
}

// MessageQueue is a free data retrieval call binding the contract method 0x29aa604b.
//
// Solidity: function messageQueue(uint256 ) view returns(bytes32)
func (_L1MessageQueue *L1MessageQueueCallerSession) MessageQueue(arg0 *big.Int) ([32]byte, error) {
	return _L1MessageQueue.Contract.MessageQueue(&_L1MessageQueue.CallOpts, arg0)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1MessageQueue *L1MessageQueueCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1MessageQueue *L1MessageQueueSession) Messenger() (common.Address, error) {
	return _L1MessageQueue.Contract.Messenger(&_L1MessageQueue.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_L1MessageQueue *L1MessageQueueCallerSession) Messenger() (common.Address, error) {
	return _L1MessageQueue.Contract.Messenger(&_L1MessageQueue.CallOpts)
}

// NextCrossDomainMessageIndex is a free data retrieval call binding the contract method 0xfd0ad31e.
//
// Solidity: function nextCrossDomainMessageIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) NextCrossDomainMessageIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "nextCrossDomainMessageIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextCrossDomainMessageIndex is a free data retrieval call binding the contract method 0xfd0ad31e.
//
// Solidity: function nextCrossDomainMessageIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) NextCrossDomainMessageIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.NextCrossDomainMessageIndex(&_L1MessageQueue.CallOpts)
}

// NextCrossDomainMessageIndex is a free data retrieval call binding the contract method 0xfd0ad31e.
//
// Solidity: function nextCrossDomainMessageIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) NextCrossDomainMessageIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.NextCrossDomainMessageIndex(&_L1MessageQueue.CallOpts)
}

// NextUnfinalizedQueueIndex is a free data retrieval call binding the contract method 0x416bdfa1.
//
// Solidity: function nextUnfinalizedQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) NextUnfinalizedQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "nextUnfinalizedQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextUnfinalizedQueueIndex is a free data retrieval call binding the contract method 0x416bdfa1.
//
// Solidity: function nextUnfinalizedQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) NextUnfinalizedQueueIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.NextUnfinalizedQueueIndex(&_L1MessageQueue.CallOpts)
}

// NextUnfinalizedQueueIndex is a free data retrieval call binding the contract method 0x416bdfa1.
//
// Solidity: function nextUnfinalizedQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) NextUnfinalizedQueueIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.NextUnfinalizedQueueIndex(&_L1MessageQueue.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1MessageQueue *L1MessageQueueCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1MessageQueue *L1MessageQueueSession) Owner() (common.Address, error) {
	return _L1MessageQueue.Contract.Owner(&_L1MessageQueue.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1MessageQueue *L1MessageQueueCallerSession) Owner() (common.Address, error) {
	return _L1MessageQueue.Contract.Owner(&_L1MessageQueue.CallOpts)
}

// PendingQueueIndex is a free data retrieval call binding the contract method 0xa85006ca.
//
// Solidity: function pendingQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCaller) PendingQueueIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "pendingQueueIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingQueueIndex is a free data retrieval call binding the contract method 0xa85006ca.
//
// Solidity: function pendingQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueSession) PendingQueueIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.PendingQueueIndex(&_L1MessageQueue.CallOpts)
}

// PendingQueueIndex is a free data retrieval call binding the contract method 0xa85006ca.
//
// Solidity: function pendingQueueIndex() view returns(uint256)
func (_L1MessageQueue *L1MessageQueueCallerSession) PendingQueueIndex() (*big.Int, error) {
	return _L1MessageQueue.Contract.PendingQueueIndex(&_L1MessageQueue.CallOpts)
}

// ScrollChain is a free data retrieval call binding the contract method 0x897630dd.
//
// Solidity: function scrollChain() view returns(address)
func (_L1MessageQueue *L1MessageQueueCaller) ScrollChain(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1MessageQueue.contract.Call(opts, &out, "scrollChain")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ScrollChain is a free data retrieval call binding the contract method 0x897630dd.
//
// Solidity: function scrollChain() view returns(address)
func (_L1MessageQueue *L1MessageQueueSession) ScrollChain() (common.Address, error) {
	return _L1MessageQueue.Contract.ScrollChain(&_L1MessageQueue.CallOpts)
}

// ScrollChain is a free data retrieval call binding the contract method 0x897630dd.
//
// Solidity: function scrollChain() view returns(address)
func (_L1MessageQueue *L1MessageQueueCallerSession) ScrollChain() (common.Address, error) {
	return _L1MessageQueue.Contract.ScrollChain(&_L1MessageQueue.CallOpts)
}

// AppendCrossDomainMessage is a paid mutator transaction binding the contract method 0x9b159782.
//
// Solidity: function appendCrossDomainMessage(address _target, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) AppendCrossDomainMessage(opts *bind.TransactOpts, _target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "appendCrossDomainMessage", _target, _gasLimit, _data)
}

// AppendCrossDomainMessage is a paid mutator transaction binding the contract method 0x9b159782.
//
// Solidity: function appendCrossDomainMessage(address _target, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueSession) AppendCrossDomainMessage(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.AppendCrossDomainMessage(&_L1MessageQueue.TransactOpts, _target, _gasLimit, _data)
}

// AppendCrossDomainMessage is a paid mutator transaction binding the contract method 0x9b159782.
//
// Solidity: function appendCrossDomainMessage(address _target, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) AppendCrossDomainMessage(_target common.Address, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.AppendCrossDomainMessage(&_L1MessageQueue.TransactOpts, _target, _gasLimit, _data)
}

// AppendEnforcedTransaction is a paid mutator transaction binding the contract method 0xbdc6f0a0.
//
// Solidity: function appendEnforcedTransaction(address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) AppendEnforcedTransaction(opts *bind.TransactOpts, _sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "appendEnforcedTransaction", _sender, _target, _value, _gasLimit, _data)
}

// AppendEnforcedTransaction is a paid mutator transaction binding the contract method 0xbdc6f0a0.
//
// Solidity: function appendEnforcedTransaction(address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueSession) AppendEnforcedTransaction(_sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.AppendEnforcedTransaction(&_L1MessageQueue.TransactOpts, _sender, _target, _value, _gasLimit, _data)
}

// AppendEnforcedTransaction is a paid mutator transaction binding the contract method 0xbdc6f0a0.
//
// Solidity: function appendEnforcedTransaction(address _sender, address _target, uint256 _value, uint256 _gasLimit, bytes _data) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) AppendEnforcedTransaction(_sender common.Address, _target common.Address, _value *big.Int, _gasLimit *big.Int, _data []byte) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.AppendEnforcedTransaction(&_L1MessageQueue.TransactOpts, _sender, _target, _value, _gasLimit, _data)
}

// DropCrossDomainMessage is a paid mutator transaction binding the contract method 0x91652461.
//
// Solidity: function dropCrossDomainMessage(uint256 _index) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) DropCrossDomainMessage(opts *bind.TransactOpts, _index *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "dropCrossDomainMessage", _index)
}

// DropCrossDomainMessage is a paid mutator transaction binding the contract method 0x91652461.
//
// Solidity: function dropCrossDomainMessage(uint256 _index) returns()
func (_L1MessageQueue *L1MessageQueueSession) DropCrossDomainMessage(_index *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.DropCrossDomainMessage(&_L1MessageQueue.TransactOpts, _index)
}

// DropCrossDomainMessage is a paid mutator transaction binding the contract method 0x91652461.
//
// Solidity: function dropCrossDomainMessage(uint256 _index) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) DropCrossDomainMessage(_index *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.DropCrossDomainMessage(&_L1MessageQueue.TransactOpts, _index)
}

// FinalizePoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x38050fd4.
//
// Solidity: function finalizePoppedCrossDomainMessage(uint256 _newFinalizedQueueIndexPlusOne) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) FinalizePoppedCrossDomainMessage(opts *bind.TransactOpts, _newFinalizedQueueIndexPlusOne *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "finalizePoppedCrossDomainMessage", _newFinalizedQueueIndexPlusOne)
}

// FinalizePoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x38050fd4.
//
// Solidity: function finalizePoppedCrossDomainMessage(uint256 _newFinalizedQueueIndexPlusOne) returns()
func (_L1MessageQueue *L1MessageQueueSession) FinalizePoppedCrossDomainMessage(_newFinalizedQueueIndexPlusOne *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.FinalizePoppedCrossDomainMessage(&_L1MessageQueue.TransactOpts, _newFinalizedQueueIndexPlusOne)
}

// FinalizePoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x38050fd4.
//
// Solidity: function finalizePoppedCrossDomainMessage(uint256 _newFinalizedQueueIndexPlusOne) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) FinalizePoppedCrossDomainMessage(_newFinalizedQueueIndexPlusOne *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.FinalizePoppedCrossDomainMessage(&_L1MessageQueue.TransactOpts, _newFinalizedQueueIndexPlusOne)
}

// Initialize is a paid mutator transaction binding the contract method 0xf7013ef6.
//
// Solidity: function initialize(address _messenger, address _scrollChain, address _enforcedTxGateway, address _gasOracle, uint256 _maxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) Initialize(opts *bind.TransactOpts, _messenger common.Address, _scrollChain common.Address, _enforcedTxGateway common.Address, _gasOracle common.Address, _maxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "initialize", _messenger, _scrollChain, _enforcedTxGateway, _gasOracle, _maxGasLimit)
}

// Initialize is a paid mutator transaction binding the contract method 0xf7013ef6.
//
// Solidity: function initialize(address _messenger, address _scrollChain, address _enforcedTxGateway, address _gasOracle, uint256 _maxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueSession) Initialize(_messenger common.Address, _scrollChain common.Address, _enforcedTxGateway common.Address, _gasOracle common.Address, _maxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.Initialize(&_L1MessageQueue.TransactOpts, _messenger, _scrollChain, _enforcedTxGateway, _gasOracle, _maxGasLimit)
}

// Initialize is a paid mutator transaction binding the contract method 0xf7013ef6.
//
// Solidity: function initialize(address _messenger, address _scrollChain, address _enforcedTxGateway, address _gasOracle, uint256 _maxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) Initialize(_messenger common.Address, _scrollChain common.Address, _enforcedTxGateway common.Address, _gasOracle common.Address, _maxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.Initialize(&_L1MessageQueue.TransactOpts, _messenger, _scrollChain, _enforcedTxGateway, _gasOracle, _maxGasLimit)
}

// PopCrossDomainMessage is a paid mutator transaction binding the contract method 0x55f613ce.
//
// Solidity: function popCrossDomainMessage(uint256 _startIndex, uint256 _count, uint256 _skippedBitmap) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) PopCrossDomainMessage(opts *bind.TransactOpts, _startIndex *big.Int, _count *big.Int, _skippedBitmap *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "popCrossDomainMessage", _startIndex, _count, _skippedBitmap)
}

// PopCrossDomainMessage is a paid mutator transaction binding the contract method 0x55f613ce.
//
// Solidity: function popCrossDomainMessage(uint256 _startIndex, uint256 _count, uint256 _skippedBitmap) returns()
func (_L1MessageQueue *L1MessageQueueSession) PopCrossDomainMessage(_startIndex *big.Int, _count *big.Int, _skippedBitmap *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.PopCrossDomainMessage(&_L1MessageQueue.TransactOpts, _startIndex, _count, _skippedBitmap)
}

// PopCrossDomainMessage is a paid mutator transaction binding the contract method 0x55f613ce.
//
// Solidity: function popCrossDomainMessage(uint256 _startIndex, uint256 _count, uint256 _skippedBitmap) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) PopCrossDomainMessage(_startIndex *big.Int, _count *big.Int, _skippedBitmap *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.PopCrossDomainMessage(&_L1MessageQueue.TransactOpts, _startIndex, _count, _skippedBitmap)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1MessageQueue *L1MessageQueueTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1MessageQueue *L1MessageQueueSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1MessageQueue.Contract.RenounceOwnership(&_L1MessageQueue.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1MessageQueue.Contract.RenounceOwnership(&_L1MessageQueue.TransactOpts)
}

// ResetPoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x7a6e9333.
//
// Solidity: function resetPoppedCrossDomainMessage(uint256 _startIndex) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) ResetPoppedCrossDomainMessage(opts *bind.TransactOpts, _startIndex *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "resetPoppedCrossDomainMessage", _startIndex)
}

// ResetPoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x7a6e9333.
//
// Solidity: function resetPoppedCrossDomainMessage(uint256 _startIndex) returns()
func (_L1MessageQueue *L1MessageQueueSession) ResetPoppedCrossDomainMessage(_startIndex *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.ResetPoppedCrossDomainMessage(&_L1MessageQueue.TransactOpts, _startIndex)
}

// ResetPoppedCrossDomainMessage is a paid mutator transaction binding the contract method 0x7a6e9333.
//
// Solidity: function resetPoppedCrossDomainMessage(uint256 _startIndex) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) ResetPoppedCrossDomainMessage(_startIndex *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.ResetPoppedCrossDomainMessage(&_L1MessageQueue.TransactOpts, _startIndex)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1MessageQueue *L1MessageQueueSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.TransferOwnership(&_L1MessageQueue.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.TransferOwnership(&_L1MessageQueue.TransactOpts, newOwner)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) UpdateGasOracle(opts *bind.TransactOpts, _newGasOracle common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "updateGasOracle", _newGasOracle)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1MessageQueue *L1MessageQueueSession) UpdateGasOracle(_newGasOracle common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.UpdateGasOracle(&_L1MessageQueue.TransactOpts, _newGasOracle)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) UpdateGasOracle(_newGasOracle common.Address) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.UpdateGasOracle(&_L1MessageQueue.TransactOpts, _newGasOracle)
}

// UpdateMaxGasLimit is a paid mutator transaction binding the contract method 0xd5ad4a97.
//
// Solidity: function updateMaxGasLimit(uint256 _newMaxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueTransactor) UpdateMaxGasLimit(opts *bind.TransactOpts, _newMaxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.contract.Transact(opts, "updateMaxGasLimit", _newMaxGasLimit)
}

// UpdateMaxGasLimit is a paid mutator transaction binding the contract method 0xd5ad4a97.
//
// Solidity: function updateMaxGasLimit(uint256 _newMaxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueSession) UpdateMaxGasLimit(_newMaxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.UpdateMaxGasLimit(&_L1MessageQueue.TransactOpts, _newMaxGasLimit)
}

// UpdateMaxGasLimit is a paid mutator transaction binding the contract method 0xd5ad4a97.
//
// Solidity: function updateMaxGasLimit(uint256 _newMaxGasLimit) returns()
func (_L1MessageQueue *L1MessageQueueTransactorSession) UpdateMaxGasLimit(_newMaxGasLimit *big.Int) (*types.Transaction, error) {
	return _L1MessageQueue.Contract.UpdateMaxGasLimit(&_L1MessageQueue.TransactOpts, _newMaxGasLimit)
}

// L1MessageQueueDequeueTransactionIterator is returned from FilterDequeueTransaction and is used to iterate over the raw logs and unpacked data for DequeueTransaction events raised by the L1MessageQueue contract.
type L1MessageQueueDequeueTransactionIterator struct {
	Event *L1MessageQueueDequeueTransaction // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueDequeueTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueDequeueTransaction)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueDequeueTransaction)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueDequeueTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueDequeueTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueDequeueTransaction represents a DequeueTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueDequeueTransaction struct {
	StartIndex    *big.Int
	Count         *big.Int
	SkippedBitmap *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterDequeueTransaction is a free log retrieval operation binding the contract event 0xc77f792f838ae38399ac31acc3348389aeb110ce7bedf3cfdbdd5e6679267970.
//
// Solidity: event DequeueTransaction(uint256 startIndex, uint256 count, uint256 skippedBitmap)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterDequeueTransaction(opts *bind.FilterOpts) (*L1MessageQueueDequeueTransactionIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "DequeueTransaction")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueDequeueTransactionIterator{contract: _L1MessageQueue.contract, event: "DequeueTransaction", logs: logs, sub: sub}, nil
}

// WatchDequeueTransaction is a free log subscription operation binding the contract event 0xc77f792f838ae38399ac31acc3348389aeb110ce7bedf3cfdbdd5e6679267970.
//
// Solidity: event DequeueTransaction(uint256 startIndex, uint256 count, uint256 skippedBitmap)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchDequeueTransaction(opts *bind.WatchOpts, sink chan<- *L1MessageQueueDequeueTransaction) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "DequeueTransaction")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueDequeueTransaction)
				if err := _L1MessageQueue.contract.UnpackLog(event, "DequeueTransaction", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDequeueTransaction is a log parse operation binding the contract event 0xc77f792f838ae38399ac31acc3348389aeb110ce7bedf3cfdbdd5e6679267970.
//
// Solidity: event DequeueTransaction(uint256 startIndex, uint256 count, uint256 skippedBitmap)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseDequeueTransaction(log types.Log) (*L1MessageQueueDequeueTransaction, error) {
	event := new(L1MessageQueueDequeueTransaction)
	if err := _L1MessageQueue.contract.UnpackLog(event, "DequeueTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueDropTransactionIterator is returned from FilterDropTransaction and is used to iterate over the raw logs and unpacked data for DropTransaction events raised by the L1MessageQueue contract.
type L1MessageQueueDropTransactionIterator struct {
	Event *L1MessageQueueDropTransaction // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueDropTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueDropTransaction)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueDropTransaction)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueDropTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueDropTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueDropTransaction represents a DropTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueDropTransaction struct {
	Index *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterDropTransaction is a free log retrieval operation binding the contract event 0x43a375005206d20a83abc71722cba68c24434a8dc1f583775be7c3fde0396cbf.
//
// Solidity: event DropTransaction(uint256 index)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterDropTransaction(opts *bind.FilterOpts) (*L1MessageQueueDropTransactionIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "DropTransaction")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueDropTransactionIterator{contract: _L1MessageQueue.contract, event: "DropTransaction", logs: logs, sub: sub}, nil
}

// WatchDropTransaction is a free log subscription operation binding the contract event 0x43a375005206d20a83abc71722cba68c24434a8dc1f583775be7c3fde0396cbf.
//
// Solidity: event DropTransaction(uint256 index)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchDropTransaction(opts *bind.WatchOpts, sink chan<- *L1MessageQueueDropTransaction) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "DropTransaction")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueDropTransaction)
				if err := _L1MessageQueue.contract.UnpackLog(event, "DropTransaction", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDropTransaction is a log parse operation binding the contract event 0x43a375005206d20a83abc71722cba68c24434a8dc1f583775be7c3fde0396cbf.
//
// Solidity: event DropTransaction(uint256 index)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseDropTransaction(log types.Log) (*L1MessageQueueDropTransaction, error) {
	event := new(L1MessageQueueDropTransaction)
	if err := _L1MessageQueue.contract.UnpackLog(event, "DropTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueFinalizedDequeuedTransactionIterator is returned from FilterFinalizedDequeuedTransaction and is used to iterate over the raw logs and unpacked data for FinalizedDequeuedTransaction events raised by the L1MessageQueue contract.
type L1MessageQueueFinalizedDequeuedTransactionIterator struct {
	Event *L1MessageQueueFinalizedDequeuedTransaction // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueFinalizedDequeuedTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueFinalizedDequeuedTransaction)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueFinalizedDequeuedTransaction)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueFinalizedDequeuedTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueFinalizedDequeuedTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueFinalizedDequeuedTransaction represents a FinalizedDequeuedTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueFinalizedDequeuedTransaction struct {
	FinalizedIndex *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterFinalizedDequeuedTransaction is a free log retrieval operation binding the contract event 0xbbbf2de085aff601d965315326f9908eb5ebbb3d1b307e7e5ec42384e3320a10.
//
// Solidity: event FinalizedDequeuedTransaction(uint256 finalizedIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterFinalizedDequeuedTransaction(opts *bind.FilterOpts) (*L1MessageQueueFinalizedDequeuedTransactionIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "FinalizedDequeuedTransaction")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueFinalizedDequeuedTransactionIterator{contract: _L1MessageQueue.contract, event: "FinalizedDequeuedTransaction", logs: logs, sub: sub}, nil
}

// WatchFinalizedDequeuedTransaction is a free log subscription operation binding the contract event 0xbbbf2de085aff601d965315326f9908eb5ebbb3d1b307e7e5ec42384e3320a10.
//
// Solidity: event FinalizedDequeuedTransaction(uint256 finalizedIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchFinalizedDequeuedTransaction(opts *bind.WatchOpts, sink chan<- *L1MessageQueueFinalizedDequeuedTransaction) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "FinalizedDequeuedTransaction")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueFinalizedDequeuedTransaction)
				if err := _L1MessageQueue.contract.UnpackLog(event, "FinalizedDequeuedTransaction", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseFinalizedDequeuedTransaction is a log parse operation binding the contract event 0xbbbf2de085aff601d965315326f9908eb5ebbb3d1b307e7e5ec42384e3320a10.
//
// Solidity: event FinalizedDequeuedTransaction(uint256 finalizedIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseFinalizedDequeuedTransaction(log types.Log) (*L1MessageQueueFinalizedDequeuedTransaction, error) {
	event := new(L1MessageQueueFinalizedDequeuedTransaction)
	if err := _L1MessageQueue.contract.UnpackLog(event, "FinalizedDequeuedTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L1MessageQueue contract.
type L1MessageQueueInitializedIterator struct {
	Event *L1MessageQueueInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueInitialized represents a Initialized event raised by the L1MessageQueue contract.
type L1MessageQueueInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterInitialized(opts *bind.FilterOpts) (*L1MessageQueueInitializedIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueInitializedIterator{contract: _L1MessageQueue.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L1MessageQueueInitialized) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueInitialized)
				if err := _L1MessageQueue.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseInitialized(log types.Log) (*L1MessageQueueInitialized, error) {
	event := new(L1MessageQueueInitialized)
	if err := _L1MessageQueue.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L1MessageQueue contract.
type L1MessageQueueOwnershipTransferredIterator struct {
	Event *L1MessageQueueOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueOwnershipTransferred represents a OwnershipTransferred event raised by the L1MessageQueue contract.
type L1MessageQueueOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L1MessageQueueOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueOwnershipTransferredIterator{contract: _L1MessageQueue.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L1MessageQueueOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueOwnershipTransferred)
				if err := _L1MessageQueue.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseOwnershipTransferred(log types.Log) (*L1MessageQueueOwnershipTransferred, error) {
	event := new(L1MessageQueueOwnershipTransferred)
	if err := _L1MessageQueue.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueQueueTransactionIterator is returned from FilterQueueTransaction and is used to iterate over the raw logs and unpacked data for QueueTransaction events raised by the L1MessageQueue contract.
type L1MessageQueueQueueTransactionIterator struct {
	Event *L1MessageQueueQueueTransaction // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueQueueTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueQueueTransaction)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueQueueTransaction)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueQueueTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueQueueTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueQueueTransaction represents a QueueTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueQueueTransaction struct {
	Sender     common.Address
	Target     common.Address
	Value      *big.Int
	QueueIndex uint64
	GasLimit   *big.Int
	Data       []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterQueueTransaction is a free log retrieval operation binding the contract event 0x69cfcb8e6d4192b8aba9902243912587f37e550d75c1fa801491fce26717f37e.
//
// Solidity: event QueueTransaction(address indexed sender, address indexed target, uint256 value, uint64 queueIndex, uint256 gasLimit, bytes data)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterQueueTransaction(opts *bind.FilterOpts, sender []common.Address, target []common.Address) (*L1MessageQueueQueueTransactionIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "QueueTransaction", senderRule, targetRule)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueQueueTransactionIterator{contract: _L1MessageQueue.contract, event: "QueueTransaction", logs: logs, sub: sub}, nil
}

// WatchQueueTransaction is a free log subscription operation binding the contract event 0x69cfcb8e6d4192b8aba9902243912587f37e550d75c1fa801491fce26717f37e.
//
// Solidity: event QueueTransaction(address indexed sender, address indexed target, uint256 value, uint64 queueIndex, uint256 gasLimit, bytes data)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchQueueTransaction(opts *bind.WatchOpts, sink chan<- *L1MessageQueueQueueTransaction, sender []common.Address, target []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "QueueTransaction", senderRule, targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueQueueTransaction)
				if err := _L1MessageQueue.contract.UnpackLog(event, "QueueTransaction", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseQueueTransaction is a log parse operation binding the contract event 0x69cfcb8e6d4192b8aba9902243912587f37e550d75c1fa801491fce26717f37e.
//
// Solidity: event QueueTransaction(address indexed sender, address indexed target, uint256 value, uint64 queueIndex, uint256 gasLimit, bytes data)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseQueueTransaction(log types.Log) (*L1MessageQueueQueueTransaction, error) {
	event := new(L1MessageQueueQueueTransaction)
	if err := _L1MessageQueue.contract.UnpackLog(event, "QueueTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueResetDequeuedTransactionIterator is returned from FilterResetDequeuedTransaction and is used to iterate over the raw logs and unpacked data for ResetDequeuedTransaction events raised by the L1MessageQueue contract.
type L1MessageQueueResetDequeuedTransactionIterator struct {
	Event *L1MessageQueueResetDequeuedTransaction // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueResetDequeuedTransactionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueResetDequeuedTransaction)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueResetDequeuedTransaction)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueResetDequeuedTransactionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueResetDequeuedTransactionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueResetDequeuedTransaction represents a ResetDequeuedTransaction event raised by the L1MessageQueue contract.
type L1MessageQueueResetDequeuedTransaction struct {
	StartIndex *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterResetDequeuedTransaction is a free log retrieval operation binding the contract event 0xc079f1a662217305bfe03e0a85f03944a2ac422f5ee5431c98b9ef7d3c6226c9.
//
// Solidity: event ResetDequeuedTransaction(uint256 startIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterResetDequeuedTransaction(opts *bind.FilterOpts) (*L1MessageQueueResetDequeuedTransactionIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "ResetDequeuedTransaction")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueResetDequeuedTransactionIterator{contract: _L1MessageQueue.contract, event: "ResetDequeuedTransaction", logs: logs, sub: sub}, nil
}

// WatchResetDequeuedTransaction is a free log subscription operation binding the contract event 0xc079f1a662217305bfe03e0a85f03944a2ac422f5ee5431c98b9ef7d3c6226c9.
//
// Solidity: event ResetDequeuedTransaction(uint256 startIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchResetDequeuedTransaction(opts *bind.WatchOpts, sink chan<- *L1MessageQueueResetDequeuedTransaction) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "ResetDequeuedTransaction")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueResetDequeuedTransaction)
				if err := _L1MessageQueue.contract.UnpackLog(event, "ResetDequeuedTransaction", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseResetDequeuedTransaction is a log parse operation binding the contract event 0xc079f1a662217305bfe03e0a85f03944a2ac422f5ee5431c98b9ef7d3c6226c9.
//
// Solidity: event ResetDequeuedTransaction(uint256 startIndex)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseResetDequeuedTransaction(log types.Log) (*L1MessageQueueResetDequeuedTransaction, error) {
	event := new(L1MessageQueueResetDequeuedTransaction)
	if err := _L1MessageQueue.contract.UnpackLog(event, "ResetDequeuedTransaction", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueUpdateGasOracleIterator is returned from FilterUpdateGasOracle and is used to iterate over the raw logs and unpacked data for UpdateGasOracle events raised by the L1MessageQueue contract.
type L1MessageQueueUpdateGasOracleIterator struct {
	Event *L1MessageQueueUpdateGasOracle // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueUpdateGasOracleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueUpdateGasOracle)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueUpdateGasOracle)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueUpdateGasOracleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueUpdateGasOracleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueUpdateGasOracle represents a UpdateGasOracle event raised by the L1MessageQueue contract.
type L1MessageQueueUpdateGasOracle struct {
	OldGasOracle common.Address
	NewGasOracle common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUpdateGasOracle is a free log retrieval operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address indexed _oldGasOracle, address indexed _newGasOracle)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterUpdateGasOracle(opts *bind.FilterOpts, _oldGasOracle []common.Address, _newGasOracle []common.Address) (*L1MessageQueueUpdateGasOracleIterator, error) {

	var _oldGasOracleRule []interface{}
	for _, _oldGasOracleItem := range _oldGasOracle {
		_oldGasOracleRule = append(_oldGasOracleRule, _oldGasOracleItem)
	}
	var _newGasOracleRule []interface{}
	for _, _newGasOracleItem := range _newGasOracle {
		_newGasOracleRule = append(_newGasOracleRule, _newGasOracleItem)
	}

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "UpdateGasOracle", _oldGasOracleRule, _newGasOracleRule)
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueUpdateGasOracleIterator{contract: _L1MessageQueue.contract, event: "UpdateGasOracle", logs: logs, sub: sub}, nil
}

// WatchUpdateGasOracle is a free log subscription operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address indexed _oldGasOracle, address indexed _newGasOracle)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchUpdateGasOracle(opts *bind.WatchOpts, sink chan<- *L1MessageQueueUpdateGasOracle, _oldGasOracle []common.Address, _newGasOracle []common.Address) (event.Subscription, error) {

	var _oldGasOracleRule []interface{}
	for _, _oldGasOracleItem := range _oldGasOracle {
		_oldGasOracleRule = append(_oldGasOracleRule, _oldGasOracleItem)
	}
	var _newGasOracleRule []interface{}
	for _, _newGasOracleItem := range _newGasOracle {
		_newGasOracleRule = append(_newGasOracleRule, _newGasOracleItem)
	}

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "UpdateGasOracle", _oldGasOracleRule, _newGasOracleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueUpdateGasOracle)
				if err := _L1MessageQueue.contract.UnpackLog(event, "UpdateGasOracle", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpdateGasOracle is a log parse operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address indexed _oldGasOracle, address indexed _newGasOracle)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseUpdateGasOracle(log types.Log) (*L1MessageQueueUpdateGasOracle, error) {
	event := new(L1MessageQueueUpdateGasOracle)
	if err := _L1MessageQueue.contract.UnpackLog(event, "UpdateGasOracle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1MessageQueueUpdateMaxGasLimitIterator is returned from FilterUpdateMaxGasLimit and is used to iterate over the raw logs and unpacked data for UpdateMaxGasLimit events raised by the L1MessageQueue contract.
type L1MessageQueueUpdateMaxGasLimitIterator struct {
	Event *L1MessageQueueUpdateMaxGasLimit // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *L1MessageQueueUpdateMaxGasLimitIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1MessageQueueUpdateMaxGasLimit)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(L1MessageQueueUpdateMaxGasLimit)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *L1MessageQueueUpdateMaxGasLimitIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1MessageQueueUpdateMaxGasLimitIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1MessageQueueUpdateMaxGasLimit represents a UpdateMaxGasLimit event raised by the L1MessageQueue contract.
type L1MessageQueueUpdateMaxGasLimit struct {
	OldMaxGasLimit *big.Int
	NewMaxGasLimit *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterUpdateMaxGasLimit is a free log retrieval operation binding the contract event 0xa030881e03ff723954dd0d35500564afab9603555d09d4456a32436f2b2373c5.
//
// Solidity: event UpdateMaxGasLimit(uint256 _oldMaxGasLimit, uint256 _newMaxGasLimit)
func (_L1MessageQueue *L1MessageQueueFilterer) FilterUpdateMaxGasLimit(opts *bind.FilterOpts) (*L1MessageQueueUpdateMaxGasLimitIterator, error) {

	logs, sub, err := _L1MessageQueue.contract.FilterLogs(opts, "UpdateMaxGasLimit")
	if err != nil {
		return nil, err
	}
	return &L1MessageQueueUpdateMaxGasLimitIterator{contract: _L1MessageQueue.contract, event: "UpdateMaxGasLimit", logs: logs, sub: sub}, nil
}

// WatchUpdateMaxGasLimit is a free log subscription operation binding the contract event 0xa030881e03ff723954dd0d35500564afab9603555d09d4456a32436f2b2373c5.
//
// Solidity: event UpdateMaxGasLimit(uint256 _oldMaxGasLimit, uint256 _newMaxGasLimit)
func (_L1MessageQueue *L1MessageQueueFilterer) WatchUpdateMaxGasLimit(opts *bind.WatchOpts, sink chan<- *L1MessageQueueUpdateMaxGasLimit) (event.Subscription, error) {

	logs, sub, err := _L1MessageQueue.contract.WatchLogs(opts, "UpdateMaxGasLimit")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1MessageQueueUpdateMaxGasLimit)
				if err := _L1MessageQueue.contract.UnpackLog(event, "UpdateMaxGasLimit", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUpdateMaxGasLimit is a log parse operation binding the contract event 0xa030881e03ff723954dd0d35500564afab9603555d09d4456a32436f2b2373c5.
//
// Solidity: event UpdateMaxGasLimit(uint256 _oldMaxGasLimit, uint256 _newMaxGasLimit)
func (_L1MessageQueue *L1MessageQueueFilterer) ParseUpdateMaxGasLimit(log types.Log) (*L1MessageQueueUpdateMaxGasLimit, error) {
	event := new(L1MessageQueueUpdateMaxGasLimit)
	if err := _L1MessageQueue.contract.UnpackLog(event, "UpdateMaxGasLimit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
