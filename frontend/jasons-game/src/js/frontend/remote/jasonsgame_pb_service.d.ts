// package: jasonsgame
// file: jasonsgame.proto

import * as jasonsgame_pb from "./jasonsgame_pb";
import {grpc} from "@improbable-eng/grpc-web";

type GameServiceSendCommand = {
  readonly methodName: string;
  readonly service: typeof GameService;
  readonly requestStream: false;
  readonly responseStream: false;
  readonly requestType: typeof jasonsgame_pb.UserInput;
  readonly responseType: typeof jasonsgame_pb.CommandReceived;
};

type GameServiceReceiveUserMessages = {
  readonly methodName: string;
  readonly service: typeof GameService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof jasonsgame_pb.Session;
  readonly responseType: typeof jasonsgame_pb.MessageToUser;
};

type GameServiceReceiveStatMessages = {
  readonly methodName: string;
  readonly service: typeof GameService;
  readonly requestStream: false;
  readonly responseStream: true;
  readonly requestType: typeof jasonsgame_pb.Session;
  readonly responseType: typeof jasonsgame_pb.Stats;
};

export class GameService {
  static readonly serviceName: string;
  static readonly SendCommand: GameServiceSendCommand;
  static readonly ReceiveUserMessages: GameServiceReceiveUserMessages;
  static readonly ReceiveStatMessages: GameServiceReceiveStatMessages;
}

export type ServiceError = { message: string, code: number; metadata: grpc.Metadata }
export type Status = { details: string, code: number; metadata: grpc.Metadata }

interface UnaryResponse {
  cancel(): void;
}
interface ResponseStream<T> {
  cancel(): void;
  on(type: 'data', handler: (message: T) => void): ResponseStream<T>;
  on(type: 'end', handler: () => void): ResponseStream<T>;
  on(type: 'status', handler: (status: Status) => void): ResponseStream<T>;
}
interface RequestStream<T> {
  write(message: T): RequestStream<T>;
  end(): void;
  cancel(): void;
  on(type: 'end', handler: () => void): RequestStream<T>;
  on(type: 'status', handler: (status: Status) => void): RequestStream<T>;
}
interface BidirectionalStream<ReqT, ResT> {
  write(message: ReqT): BidirectionalStream<ReqT, ResT>;
  end(): void;
  cancel(): void;
  on(type: 'data', handler: (message: ResT) => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'end', handler: () => void): BidirectionalStream<ReqT, ResT>;
  on(type: 'status', handler: (status: Status) => void): BidirectionalStream<ReqT, ResT>;
}

export class GameServiceClient {
  readonly serviceHost: string;

  constructor(serviceHost: string, options?: grpc.RpcOptions);
  sendCommand(
    requestMessage: jasonsgame_pb.UserInput,
    metadata: grpc.Metadata,
    callback: (error: ServiceError|null, responseMessage: jasonsgame_pb.CommandReceived|null) => void
  ): UnaryResponse;
  sendCommand(
    requestMessage: jasonsgame_pb.UserInput,
    callback: (error: ServiceError|null, responseMessage: jasonsgame_pb.CommandReceived|null) => void
  ): UnaryResponse;
  receiveUserMessages(requestMessage: jasonsgame_pb.Session, metadata?: grpc.Metadata): ResponseStream<jasonsgame_pb.MessageToUser>;
  receiveStatMessages(requestMessage: jasonsgame_pb.Session, metadata?: grpc.Metadata): ResponseStream<jasonsgame_pb.Stats>;
}
