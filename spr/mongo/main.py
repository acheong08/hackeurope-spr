import json
import uuid
from datetime import datetime
from typing import Generator
from typing import Optional
from datetime import datetime
from fastapi import FastAPI, UploadFile, File, HTTPException
from pymongo import MongoClient, ASCENDING
from pymongo.errors import BulkWriteError
from collections import defaultdict

import os

app = FastAPI(title="Tracee Ingestion Service")

# ---- MongoDB Configuration ----
import os
from pymongo import MongoClient

MONGO_URI = os.getenv("MONGO_URI", "mongodb://mongo:27017")
DB_NAME = os.getenv("DB_NAME", "tracee_analysis")

client = MongoClient(MONGO_URI)
db = client[DB_NAME]


# ---- Helper: Stream JSON Lines ----
def stream_json_lines(file) -> Generator[dict, None, None]:
    """
    Stream JSON objects line-by-line from uploaded file.
    Skips invalid JSON lines.
    """
    for line in file:
        try:
            yield json.loads(line)
        except json.JSONDecodeError:
            continue


# ---- Endpoint: Upload Tracee File ----
@app.post("/upload-tracee")
async def upload_tracee(file: UploadFile = File(...)):
    """
    Upload a Tracee JSONL file.
    Creates a new collection for each upload.
    Inserts every trace event as its own document.
    """

    allowed_extensions = (".json", ".jsonl", ".log")

    if not file.filename.lower().endswith(allowed_extensions):
        raise HTTPException(
            status_code=400,
            detail="File must be .json, .jsonl, or .log"
        )    # Create unique collection name

    collection_name = f"tracee_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}_{uuid.uuid4().hex[:6]}"
    collection = db[collection_name]

    # Optional: create indexes for common trace queries
    collection.create_index([("timestamp", ASCENDING)])
    collection.create_index([("processId", ASCENDING)])
    collection.create_index([("eventName", ASCENDING)])
    collection.create_index([("container.id", ASCENDING)])

    batch_size = 1000
    batch = []
    inserted_count = 0

    try:
        while True:
            chunk = await file.read(1024 * 1024)  # 1MB chunk
            if not chunk:
                break

            lines = chunk.decode("utf-8").splitlines()

            for line in lines:
                try:
                    event = json.loads(line)
                    batch.append(event)
                except json.JSONDecodeError:
                    continue

                if len(batch) >= batch_size:
                    collection.insert_many(batch)
                    inserted_count += len(batch)
                    batch = []

        # Insert remaining batch
        if batch:
            collection.insert_many(batch)
            inserted_count += len(batch)

    except BulkWriteError as e:
        raise HTTPException(status_code=500, detail=str(e))

    return {
        "status": "success",
        "collection": collection_name,
        "documents_inserted": inserted_count
    }

SENSITIVE_PATHS = [
    "/etc/passwd",
    "/etc/shadow",
    "/root",
    ".ssh",
]

SHELL_BINARIES = [
    "/bin/sh",
    "/bin/bash",
    "sh",
    "bash"
]


@app.get("/stats/{collection_name}")
def get_stats(collection_name: str):
    if collection_name not in db.list_collection_names():
        raise HTTPException(status_code=404, detail="Collection not found")

    collection = db[collection_name]

    # -------------------------
    # Total events
    # -------------------------
    total_events = collection.count_documents({})

    # -------------------------
    # Syscall frequency
    # -------------------------
    syscall_pipeline = [
        {"$group": {"_id": "$eventName", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}}
    ]
    syscall_profile = {
        item["_id"]: item["count"]
        for item in collection.aggregate(syscall_pipeline)
    }

    # -------------------------
    # File access (paths + counts)
    # -------------------------
    file_pipeline = [
        {"$match": {"eventName": "openat"}},
        {"$unwind": "$args"},
        {"$match": {"args.name": "pathname"}},
        {"$group": {"_id": "$args.value", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}}
    ]
    file_results = list(collection.aggregate(file_pipeline))

    file_access = {
        item["_id"]: item["count"]
        for item in file_results
    }

    sensitive_hits = [
        path for path in file_access
        if any(s in path for s in SENSITIVE_PATHS)
    ]

    # -------------------------
    # Executed commands (execve)
    # -------------------------
    exec_pipeline = [
        {"$match": {"eventName": "execve"}},
        {"$unwind": "$args"},
        {"$match": {"args.name": "pathname"}},
        {"$group": {"_id": "$args.value", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}}
    ]
    exec_results = list(collection.aggregate(exec_pipeline))

    executed_commands = {
        item["_id"]: item["count"]
        for item in exec_results
    }

    shell_spawned = any(
        any(shell in cmd for shell in SHELL_BINARIES)
        for cmd in executed_commands
    )

    # -------------------------
    # Network IP aggregation (SAFE VERSION)
    # -------------------------
    ip_pipeline = [
        {"$match": {"eventName": "connect"}},
        {"$unwind": "$args"},
        {"$match": {"args.name": "addr"}},
        {"$group": {"_id": "$args.value", "count": {"$sum": 1}}},
        {"$sort": {"count": -1}}
    ]

    ip_results = list(collection.aggregate(ip_pipeline))

    ip_usage = {}

    for item in ip_results:
        addr = item["_id"]

        # If Tracee stored addr as dict
        if isinstance(addr, dict):
            ip = addr.get("ip", "unknown")
            port = addr.get("port", "")
            key = f"{ip}:{port}" if port else ip
        else:
            key = str(addr)

        ip_usage[key] = item["count"]

    # -------------------------
    # DNS aggregation (CORRECT TRACEe VERSION)
    # -------------------------
    dns_pipeline = [
        {"$match": {"eventName": "net_packet_dns_request"}},
        {"$unwind": "$args"},
        {"$match": {"args.name": "dns_questions"}},
        {"$unwind": "$args.value"},
        {
            "$group": {
                "_id": "$args.value.query",
                "count": {"$sum": 1}
            }
        },
        {"$sort": {"count": -1}}
    ]

    dns_results = list(collection.aggregate(dns_pipeline))

    dns_usage = {
        item["_id"]: item["count"]
        for item in dns_results
    }
    # -------------------------
    # Risk Flags
    # -------------------------
    risk_flags = []

    if sensitive_hits:
        risk_flags.append("sensitive_file_access")

    if shell_spawned:
        risk_flags.append("shell_spawned")

    if ip_usage:
        risk_flags.append("network_activity")

    if any("/proc" in path for path in file_access):
        risk_flags.append("procfs_access")

    return {
        "collection": collection_name,
        "total_events": total_events,
        "syscall_profile": syscall_profile,
        "file_access": file_access,
        "executed_commands": executed_commands,
        "network_activity": {
            "ips": ip_usage,
            "dns_records": dns_usage
        },
        "risk_flags": risk_flags
    }# ---- Health Check ----

@app.get("/specific/{collection_name}")
def get_specific_data(
    collection_name: str,
    dns: Optional[str] = None,
    command: Optional[str] = None,
    file: Optional[str] = None,
    limit: int = 50,
    offset: int = 0,
):
    if collection_name not in db.list_collection_names():
        raise HTTPException(status_code=404, detail="Collection not found")

    collection = db[collection_name]

    results = []

    # -------------------------
    # DNS SPECIFIC QUERY
    # -------------------------
    if dns:
        query = {
            "eventName": "net_packet_dns_request",
            "args": {
                "$elemMatch": {
                    "name": "dns_questions",
                    "value": {
                        "$elemMatch": {"query": dns}
                    }
                }
            }
        }

        cursor = collection.find(query).skip(offset).limit(limit)

        for event in cursor:
            results.append({
                "process": event.get("processName"),
                "process_id": event.get("processId"),
                "executable": event.get("executable", {}).get("path"),
                "timestamp": event.get("timestamp"),
                "args": event.get("args")  # FULL ARRAY
            })

        return {"specific": {"dns_calls": {dns: results}}}

    # -------------------------
    # COMMAND SPECIFIC QUERY (FULL ARGS SAFE)
    # -------------------------
    if command:
        query = {
            "eventName": "execve",
            "args": {
                "$elemMatch": {
                    "name": "pathname",
                    "value": command
                }
            }
        }

        cursor = collection.find(query).skip(offset).limit(limit)

        for event in cursor:
            results.append({
                "process": event.get("processName"),
                "process_id": event.get("processId"),
                "parent_process_id": event.get("parentProcessId"),
                "executable": event.get("executable", {}).get("path"),
                "timestamp": event.get("timestamp"),
                "args": event.get("args")  # FULL ARGS ARRAY PRESERVED
            })

        return {"specific": {"command_calls": {command: results}}}

    # -------------------------
    # FILE ACCESS SPECIFIC QUERY
    # -------------------------
    if file:
        query = {
            "eventName": "openat",
            "args": {
                "$elemMatch": {
                    "name": "pathname",
                    "value": file
                }
            }
        }

        cursor = collection.find(query).skip(offset).limit(limit)

        for event in cursor:
            results.append({
                "process": event.get("processName"),
                "process_id": event.get("processId"),
                "executable": event.get("executable", {}).get("path"),
                "timestamp": event.get("timestamp"),
                "args": event.get("args")  # FULL ARRAY
            })

        return {"specific": {"file_access": {file: results}}}

    raise HTTPException(
        status_code=400,
        detail="Provide one of: dns, command, or file query parameter"
    )

@app.get("/health")
def health():
    return {"status": "ok"}
