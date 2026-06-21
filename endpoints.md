# API Endpoints

REST API reference for the Daily Expenses backend. All routes are prefixed with `/api`.

**Base URL:** `http://<host>:8080/api` (default port `8080`, configurable via `ADDR`)

**Content type:** `application/json` for request and response bodies.

**Errors:** Failed requests return JSON `{"error": "<message>"}` with an appropriate HTTP status code.

There is no authentication. Anyone who can reach the server can call these endpoints.

---

## Persons

### `GET /persons`

List the two fixed people in the app.

**Response `200`**

```json
[
  { "id": 1, "name": "Person 1" },
  { "id": 2, "name": "Person 2" }
]
```

---

## Shops

### `GET /shops`

List all shops, sorted by name.

**Response `200`**

```json
[
  { "id": 1, "name": "Grocery Store" }
]
```

### `POST /shops`

Create a new shop.

**Request body**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | yes | Trimmed; must be non-empty |

```json
{ "name": "Grocery Store" }
```

**Response `201`**

```json
{ "id": 3, "name": "Grocery Store" }
```

**Errors**

| Status | Condition |
|--------|-----------|
| `400` | Missing or empty `name` |
| `409` | Shop name already exists |

### `DELETE /shops/:id`

Delete a shop by ID.

**Response `204`** — no body.

**Errors**

| Status | Condition |
|--------|-----------|
| `400` | Invalid `id` |
| `404` | Shop not found |
| `409` | Shop is referenced by one or more invoices |

---

## Invoices

### `GET /invoices`

List invoices with line items, newest first.

**Query parameters** (all optional)

| Parameter | Type | Description |
|-----------|------|-------------|
| `person_id` | integer | Filter by person (`1` or `2`) |
| `from_date` | string | Include invoices on or after this date (`YYYY-MM-DD`) |
| `to_date` | string | Include invoices on or before this date (`YYYY-MM-DD`) |

**Response `200`**

```json
[
  {
    "id": 10,
    "person_id": 1,
    "person_name": "Person 1",
    "shop_id": 2,
    "shop_name": "Grocery Store",
    "date": "2026-06-15",
    "total": 24.50,
    "items": [
      {
        "id": 21,
        "invoice_id": 10,
        "description": "Milk",
        "amount": 4.50,
        "quantity": 2
      }
    ]
  }
]
```

### `GET /invoices/:id`

Get a single invoice with line items.

**Response `200`** — same shape as one element in the list response above.

**Errors**

| Status | Condition |
|--------|-----------|
| `400` | Invalid `id` |
| `404` | Invoice not found |

### `POST /invoices`

Create an invoice and its line items in one transaction. The server computes `total` as the sum of `amount × quantity` for each item. If `quantity` is omitted or `≤ 0`, it defaults to `1`.

**Request body**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `person_id` | integer | yes | Must be `1` or `2` |
| `shop_id` | integer | yes | Must reference an existing shop |
| `date` | string | yes | `YYYY-MM-DD` |
| `items` | array | yes | At least one item |
| `items[].description` | string | yes | Trimmed before save |
| `items[].amount` | number | yes | Unit price |
| `items[].quantity` | number | no | Defaults to `1` if missing or `≤ 0` |

```json
{
  "person_id": 1,
  "shop_id": 2,
  "date": "2026-06-15",
  "items": [
    { "description": "Milk", "amount": 4.50, "quantity": 2 }
  ]
}
```

**Response `201`** — full invoice object (same shape as `GET /invoices/:id`). If the created invoice cannot be loaded, the response is `{"id": <new_id>}`.

**Errors**

| Status | Condition |
|--------|-----------|
| `400` | Invalid JSON, validation failure, `person_id` not `1` or `2`, or unknown `shop_id` |

### `DELETE /invoices/:id`

Delete an invoice and all of its line items.

**Response `204`** — no body.

**Errors**

| Status | Condition |
|--------|-----------|
| `400` | Invalid `id` |
| `404` | Invoice not found |

---

## Static files and unknown routes

When a static frontend is deployed (`STATIC_DIR`), the server also serves:

- `GET /assets/*` — built frontend assets
- `GET /*` (non-API) — `index.html` for client-side routing

Unknown paths under `/api` return `404` with `{"error": "not found"}`.
