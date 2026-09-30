import assert from 'node:assert/strict';
import { once } from 'node:events';
import { createServer } from 'node:http';
import { describe, it } from 'node:test';

import { createUserService } from './main.ts';

describe('createUserService', () => {
  it('requests_user_with_fetch', async t => {
    let requestURL: string | undefined;
    const server = createServer((request, response) => {
      requestURL = request.url;
      response.setHeader('Content-Type', 'application/json');
      response.end(JSON.stringify({ user_id: '1', name: 'User 1' }));
    });
    t.after(() => {
      server.close();
    });
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    assert.ok(address && typeof address !== 'string');

    const service = createUserService(`http://127.0.0.1:${address.port}`);
    assert.deepEqual(await service.getUser({ user_id: '1' }), { user_id: '1', name: 'User 1' });
    assert.equal(requestURL, '/users/1');
  });
});
