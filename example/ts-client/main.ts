import process from 'node:process';

import { AdaptComposer, FetchClient, JsonResolver } from '@icytown/scaffold-client';

import { UserService } from './gen/user.service.ts';

export function createUserService(baseURL = 'http://localhost:3000'): UserService {
  const client = new FetchClient({ baseURL });
  const composer = new AdaptComposer();
  const resolver = new JsonResolver();
  return new UserService(client, { composer, resolver });
}

async function main() {
  const service = createUserService();
  const user = await service.getUser({ user_id: '1' });
  process.stdout.write(`${JSON.stringify(user)}\n`);
}

if (import.meta.main) {
  await main();
}
