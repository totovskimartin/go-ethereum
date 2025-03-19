require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  networks: {
    localhost: {
      url: "http://127.0.0.1:8545",
      accounts: {
        mnemonic: "test test test test test test test test test test test junk",
      },
      chainId: 1337,
      gasPrice: 150
    },
  },
  solidity: {
    version: "0.8.28",
  }
};