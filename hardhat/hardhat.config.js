require("@nomicfoundation/hardhat-toolbox");
require("@nomicfoundation/hardhat-ignition-ethers");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.28",
  networks: {
    devnet: {
      url: `http://127.0.0.1:8545`,
      gasPrice: 30000000000,
    },
  },
};