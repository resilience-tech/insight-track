const insertRecords = require("./fns/insert");
const retreveals = require("./fns/retrieve");
const generateRandomData = require("./data-generator/dynamic-random");
const resetRoutes = require('./rest-routs');

module.exports = {
  retreveals,
  insertRecords,
  generateRandomData,
  resetRoutes,
};
