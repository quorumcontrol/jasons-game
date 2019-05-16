// package: jasonsgame
// file: jasonsgame.proto

import * as jspb from "google-protobuf";

export class UserInput extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  hasSession(): boolean;
  clearSession(): void;
  getSession(): Session | undefined;
  setSession(value?: Session): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserInput.AsObject;
  static toObject(includeInstance: boolean, msg: UserInput): UserInput.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UserInput, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UserInput;
  static deserializeBinaryFromReader(message: UserInput, reader: jspb.BinaryReader): UserInput;
}

export namespace UserInput {
  export type AsObject = {
    message: string,
    session?: Session.AsObject,
  }
}

export class MessageToUser extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  hasLocation(): boolean;
  clearLocation(): void;
  getLocation(): Location | undefined;
  setLocation(value?: Location): void;

  getSequence(): number;
  setSequence(value: number): void;

  getHeartbeat(): boolean;
  setHeartbeat(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MessageToUser.AsObject;
  static toObject(includeInstance: boolean, msg: MessageToUser): MessageToUser.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MessageToUser, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MessageToUser;
  static deserializeBinaryFromReader(message: MessageToUser, reader: jspb.BinaryReader): MessageToUser;
}

export namespace MessageToUser {
  export type AsObject = {
    message: string,
    location?: Location.AsObject,
    sequence: number,
    heartbeat: boolean,
  }
}

export class Stats extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Stats.AsObject;
  static toObject(includeInstance: boolean, msg: Stats): Stats.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Stats, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Stats;
  static deserializeBinaryFromReader(message: Stats, reader: jspb.BinaryReader): Stats;
}

export namespace Stats {
  export type AsObject = {
    message: string,
  }
}

export class Exit extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Exit.AsObject;
  static toObject(includeInstance: boolean, msg: Exit): Exit.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Exit, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Exit;
  static deserializeBinaryFromReader(message: Exit, reader: jspb.BinaryReader): Exit;
}

export namespace Exit {
  export type AsObject = {
  }
}

export class Location extends jspb.Message {
  getDid(): string;
  setDid(value: string): void;

  getTip(): string;
  setTip(value: string): void;

  getX(): number;
  setX(value: number): void;

  getY(): number;
  setY(value: number): void;

  getDescription(): string;
  setDescription(value: string): void;

  hasPortal(): boolean;
  clearPortal(): void;
  getPortal(): Portal | undefined;
  setPortal(value?: Portal): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Location.AsObject;
  static toObject(includeInstance: boolean, msg: Location): Location.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Location, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Location;
  static deserializeBinaryFromReader(message: Location, reader: jspb.BinaryReader): Location;
}

export namespace Location {
  export type AsObject = {
    did: string,
    tip: string,
    x: number,
    y: number,
    description: string,
    portal?: Portal.AsObject,
  }
}

export class Player extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Player.AsObject;
  static toObject(includeInstance: boolean, msg: Player): Player.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Player, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Player;
  static deserializeBinaryFromReader(message: Player, reader: jspb.BinaryReader): Player;
}

export namespace Player {
  export type AsObject = {
    name: string,
  }
}

export class Portal extends jspb.Message {
  getTo(): string;
  setTo(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Portal.AsObject;
  static toObject(includeInstance: boolean, msg: Portal): Portal.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Portal, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Portal;
  static deserializeBinaryFromReader(message: Portal, reader: jspb.BinaryReader): Portal;
}

export namespace Portal {
  export type AsObject = {
    to: string,
  }
}

export class CommandReceived extends jspb.Message {
  getSequence(): number;
  setSequence(value: number): void;

  getError(): boolean;
  setError(value: boolean): void;

  getErrorMessage(): string;
  setErrorMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CommandReceived.AsObject;
  static toObject(includeInstance: boolean, msg: CommandReceived): CommandReceived.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CommandReceived, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CommandReceived;
  static deserializeBinaryFromReader(message: CommandReceived, reader: jspb.BinaryReader): CommandReceived;
}

export namespace CommandReceived {
  export type AsObject = {
    sequence: number,
    error: boolean,
    errorMessage: string,
  }
}

export class Session extends jspb.Message {
  getUuid(): string;
  setUuid(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Session.AsObject;
  static toObject(includeInstance: boolean, msg: Session): Session.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Session, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Session;
  static deserializeBinaryFromReader(message: Session, reader: jspb.BinaryReader): Session;
}

export namespace Session {
  export type AsObject = {
    uuid: string,
  }
}

