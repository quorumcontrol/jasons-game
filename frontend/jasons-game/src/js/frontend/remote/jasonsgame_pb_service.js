// package: jasonsgame
// file: jasonsgame.proto

var jasonsgame_pb = require("./jasonsgame_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var GameService = (function () {
  function GameService() {}
  GameService.serviceName = "jasonsgame.GameService";
  return GameService;
}());

GameService.SendCommand = {
  methodName: "SendCommand",
  service: GameService,
  requestStream: false,
  responseStream: false,
  requestType: jasonsgame_pb.jasonsgame.UserInput,
  responseType: jasonsgame_pb.jasonsgame.CommandReceived
};

GameService.ReceiveUIMessages = {
  methodName: "ReceiveUIMessages",
  service: GameService,
  requestStream: false,
  responseStream: true,
  requestType: jasonsgame_pb.jasonsgame.Session,
  responseType: jasonsgame_pb.jasonsgame.UserInterfaceMessage
};

GameService.ReceiveStatMessages = {
  methodName: "ReceiveStatMessages",
  service: GameService,
  requestStream: false,
  responseStream: true,
  requestType: jasonsgame_pb.jasonsgame.Session,
  responseType: jasonsgame_pb.jasonsgame.Stats
};

exports.GameService = GameService;

function GameServiceClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

GameServiceClient.prototype.sendCommand = function sendCommand(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(GameService.SendCommand, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

GameServiceClient.prototype.receiveUIMessages = function receiveUIMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(GameService.ReceiveUIMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

GameServiceClient.prototype.receiveStatMessages = function receiveStatMessages(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(GameService.ReceiveStatMessages, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

exports.GameServiceClient = GameServiceClient;

