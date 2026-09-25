# discount-engine

Custom Rule-Based Discount Engine


https://-.git

# Folder structure

````text
discount-engine/
├── cmd/
│ └── server/
│ └── main.go # HTTP entry point
│
├── internal/
│ ├── engine/
│ │ ├── engine.go # Core discount logic
│ │ ├── rule.go # Rule & condition models
│ │ └── evaluator.go # Rule evaluation logic
│ │
│ └── store/
│ └── rule_loader.go # Load rules from JSON
│
├── configs/
│ └── rules.json
│
├── tests/
│ └── engine_test.go
│
├── go.mod
└── README.md
```text
````
