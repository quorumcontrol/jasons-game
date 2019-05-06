/**
 * @fileoverview gRPC-Web generated client stub for 
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!



const grpc = {};
grpc.web = require('grpc-web');

const proto = require('./books_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?Object} options
 * @constructor
 * @struct
 * @final
 */
proto.BookServiceClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

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
proto.BookServicePromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options['format'] = 'text';

  /**
   * @private @const {!proto.BookServiceClient} The delegate callback based client
   */
  this.delegateClient_ = new proto.BookServiceClient(
      hostname, credentials, options);

};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.GetBookRequest,
 *   !proto.Book>}
 */
const methodInfo_BookService_GetBook = new grpc.web.AbstractClientBase.MethodInfo(
  proto.Book,
  /** @param {!proto.GetBookRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.Book.deserializeBinary
);


/**
 * @param {!proto.GetBookRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.Error, ?proto.Book)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.Book>|undefined}
 *     The XHR Node Readable Stream
 */
proto.BookServiceClient.prototype.getBook =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/BookService/GetBook',
      request,
      metadata,
      methodInfo_BookService_GetBook,
      callback);
};


/**
 * @param {!proto.GetBookRequest} request The
 *     request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.Book>}
 *     The XHR Node Readable Stream
 */
proto.BookServicePromiseClient.prototype.getBook =
    function(request, metadata) {
  return new Promise((resolve, reject) => {
    this.delegateClient_.getBook(
      request, metadata, (error, response) => {
        error ? reject(error) : resolve(response);
      });
  });
};


/**
 * @const
 * @type {!grpc.web.AbstractClientBase.MethodInfo<
 *   !proto.QueryBooksRequest,
 *   !proto.Book>}
 */
const methodInfo_BookService_QueryBooks = new grpc.web.AbstractClientBase.MethodInfo(
  proto.Book,
  /** @param {!proto.QueryBooksRequest} request */
  function(request) {
    return request.serializeBinary();
  },
  proto.Book.deserializeBinary
);


/**
 * @param {!proto.QueryBooksRequest} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.Book>}
 *     The XHR Node Readable Stream
 */
proto.BookServiceClient.prototype.queryBooks =
    function(request, metadata) {
  return this.client_.serverStreaming(this.hostname_ +
      '/BookService/QueryBooks',
      request,
      metadata,
      methodInfo_BookService_QueryBooks);
};


/**
 * @param {!proto.QueryBooksRequest} request The request proto
 * @param {!Object<string, string>} metadata User defined
 *     call metadata
 * @return {!grpc.web.ClientReadableStream<!proto.Book>}
 *     The XHR Node Readable Stream
 */
proto.BookServicePromiseClient.prototype.queryBooks =
    function(request, metadata) {
  return this.delegateClient_.client_.serverStreaming(this.delegateClient_.hostname_ +
      '/BookService/QueryBooks',
      request,
      metadata,
      methodInfo_BookService_QueryBooks);
};


module.exports = proto;

