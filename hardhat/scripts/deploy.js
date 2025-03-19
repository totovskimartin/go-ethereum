const ApolloModule = require("../ignition/modules/Apollo");

async function main() {
  const { apollo } = await hre.ignition.deploy(ApolloModule);

  console.log(`Apollo deployed to: ${await apollo.getAddress()}`);
}

main().catch(console.error);