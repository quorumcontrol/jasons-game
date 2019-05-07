/**
 * @fileoverview gRPC-Web generated client stub for jasonsgame
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!



const grpc = {};
grpc.web = require('grpc-web');


var github_com_gogo_protobuf_gogoproto_gogo_pb = require('./github.com/gogo/protobuf/gogoproto/gogo_pb.js')
const proto = {};
proto.jasonsgame = require('./jasonsgame_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.jasonsgame.GameServiceClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  /**
   * @private @const {!grpc.web.OPClientBase} The client
   */
  this.client_ = new grpc.web.OPClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

  /**
   * @private @const {?Object} The credentials to be used to connect
   *    to the server
   */
  this.credentials_ = credentials;

  /**
   * @private @const {?Object} Options for the client
   */
  this.options_ = options;
};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.jasonsgame.GameServicePromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  /**
   * @private @const {!proto.jasonsgame.GameServiceClient} The delegate callback based client
   */
  this.delegateClient_ = new proto.jasonsgame.GameServiceClient(
      hostname, credentials, options);

};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.jasonsgame.UserInput,
 *   !proto.jasonsgame.CommandReceived>}
 */
const methodInfo_GameService_SendCommand = new grpc.web.AbstractClientBase.MethodInfo(
  proto.jasonsgame.CommandReceived,
  /** @param {!proto.jasonsgame.UserInput} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.jasonsgame.CommandReceived.deserializeBinary
);


/**
 * @param {!proto.jasonsgame.UserInput} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.jasonsgame.CommandReceived)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.jasonsgame.CommandReceived>|undefined}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServiceClient.prototype.sendCommand =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/$rpc/jasonsgame.GameService/SendCommand',
      request,
      metadata,
      methodInfo_GameService_SendCommand,
      callback);
};


/**
 * @param {!proto.jasonsgame.UserInput} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.jasonsgame.CommandReceived>}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServicePromiseClient.prototype.sendCommand =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.sendCommand(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.jasonsgame.Session,
 *   !proto.jasonsgame.MessageToUser>}
 */
const methodInfo_GameService_ReceiveUserMessages = new grpc.web.AbstractClientBase.MethodInfo(
  proto.jasonsgame.MessageToUser,
  /** @param {!proto.jasonsgame.Session} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.jasonsgame.MessageToUser.deserializeBinary
);


/**
 * @param {!proto.jasonsgame.Session} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.jasonsgame.MessageToUser>}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServiceClient.prototype.receiveUserMessages =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/$rpc/jasonsgame.GameService/ReceiveUserMessages',
      request,
      metadata,
      methodInfo_GameService_ReceiveUserMessages);
};


/**
 * @param {!proto.jasonsgame.Session} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.jasonsgame.MessageToUser>}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServicePromiseClient.prototype.receiveUserMessages =
    function(request, metadata) {
  return this.delegateClient_.client_.serverStreaming(this.delegateClient_.hostname_ +
      '/$rpc/jasonsgame.GameService/ReceiveUserMessages',
      request,
      metadata,
      methodInfo_GameService_ReceiveUserMessages);
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.jasonsgame.Session,
 *   !proto.jasonsgame.Stats>}
 */
const methodInfo_GameService_ReceiveStatMessages = new grpc.web.AbstractClientBase.MethodInfo(
  proto.jasonsgame.Stats,
  /** @param {!proto.jasonsgame.Session} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.jasonsgame.Stats.deserializeBinary
);


/**
 * @param {!proto.jasonsgame.Session} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.jasonsgame.Stats>}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServiceClient.prototype.receiveStatMessages =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/$rpc/jasonsgame.GameService/ReceiveStatMessages',
      request,
      metadata,
      methodInfo_GameService_ReceiveStatMessages);
};


/**
 * @param {!proto.jasonsgame.Session} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.jasonsgame.Stats>}
 *     The XHR Node Readable Stream
 */
proto.jasonsgame.GameServicePromiseClient.prototype.receiveStatMessages =
    function(request, metadata) {
  return this.delegateClient_.client_.serverStreaming(this.delegateClient_.hostname_ +
      '/$rpc/jasonsgame.GameService/ReceiveStatMessages',
      request,
      metadata,
      methodInfo_GameService_ReceiveStatMessages);
};


module.exports = proto.jasonsgame;

