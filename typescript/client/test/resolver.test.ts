import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { HTTPError, JsonResolver, type Response } from '../src/index.ts';

void describe('JsonResolver.resolve', () => {
  void it('parses a successful JSON response', () => {
    const response = createResponse(200, '{"name":"Alice"}');

    assert.deepEqual(new JsonResolver().resolve(response), { name: 'Alice' });
  });

  void it('returns undefined for an empty successful response', () => {
    assert.equal(new JsonResolver().resolve(createResponse(204, '')), undefined);
  });

  void it('throws HTTPError with parsed response data', () => {
    const response = createResponse(404, '{"message":"not found"}');

    assert.throws(
      () => new JsonResolver().resolve(response),
      (error: unknown) => {
        assert.ok(error instanceof HTTPError);
        assert.equal(error.response, response);
        assert.deepEqual(error.data, { message: 'not found' });
        return true;
      },
    );
  });
});

function createResponse(status: number, body: string): Response {
  return {
    status,
    headers: {},
    body: new TextEncoder().encode(body),
  };
}
