"use strict";
(globalThis["webpackChunkwebsite"] = globalThis["webpackChunkwebsite"] || []).push([[443],{

/***/ 2700
(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  assets: () => (/* binding */ assets),
  contentTitle: () => (/* binding */ contentTitle),
  "default": () => (/* binding */ MDXContent),
  frontMatter: () => (/* binding */ frontMatter),
  metadata: () => (/* reexport */ site_docs_api_reference_md_964_namespaceObject),
  toc: () => (/* binding */ toc)
});

;// ./.docusaurus/docusaurus-plugin-content-docs/default/site-docs-api-reference-md-964.json
const site_docs_api_reference_md_964_namespaceObject = /*#__PURE__*/JSON.parse('{"id":"api-reference","title":"API Reference","description":"s3proxy exposes an S3-compatible HTTP API rather than a custom JSON API.","source":"@site/docs/api-reference.md","sourceDirName":".","slug":"/api-reference","permalink":"/docs/api-reference","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":5,"frontMatter":{"sidebar_position":5},"sidebar":"docsSidebar","previous":{"title":"Request Examples","permalink":"/docs/request-examples"},"next":{"title":"Operations","permalink":"/docs/operations"}}');
// EXTERNAL MODULE: ./node_modules/.pnpm/react@19.2.6/node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(1325);
// EXTERNAL MODULE: ./node_modules/.pnpm/@mdx-js+react@3.1.1_@types+react@19.2.14_react@19.2.6/node_modules/@mdx-js/react/lib/index.js
var lib = __webpack_require__(1982);
;// ./docs/api-reference.md


const frontMatter = {
	sidebar_position: 5
};
const contentTitle = 'API Reference';

const assets = {

};



const toc = [{
  "value": "Supported Operations",
  "id": "supported-operations",
  "level": 2
}, {
  "value": "Supported Query Keys",
  "id": "supported-query-keys",
  "level": 2
}, {
  "value": "Addressing Modes",
  "id": "addressing-modes",
  "level": 2
}, {
  "value": "Authentication Modes",
  "id": "authentication-modes",
  "level": 2
}, {
  "value": "Request Payload Formats",
  "id": "request-payload-formats",
  "level": 2
}, {
  "value": "Request Classification",
  "id": "request-classification",
  "level": 2
}, {
  "value": "Routing-Specific Behavior",
  "id": "routing-specific-behavior",
  "level": 2
}, {
  "value": "Streaming Responses",
  "id": "streaming-responses",
  "level": 2
}, {
  "value": "<code>ListBuckets</code>",
  "id": "listbuckets",
  "level": 2
}, {
  "value": "<code>ListObjectsV2</code>",
  "id": "listobjectsv2",
  "level": 2
}, {
  "value": "Failover Rules",
  "id": "failover-rules",
  "level": 2
}, {
  "value": "Fan-Out Writes",
  "id": "fan-out-writes",
  "level": 2
}, {
  "value": "Outbound Signing",
  "id": "outbound-signing",
  "level": 2
}, {
  "value": "Upstream Redirects",
  "id": "upstream-redirects",
  "level": 3
}, {
  "value": "Error Behavior",
  "id": "error-behavior",
  "level": 2
}, {
  "value": "Health Endpoints",
  "id": "health-endpoints",
  "level": 2
}];
function _createMdxContent(props) {
  const _components = {
    a: "a",
    code: "code",
    h1: "h1",
    h2: "h2",
    h3: "h3",
    header: "header",
    li: "li",
    p: "p",
    strong: "strong",
    table: "table",
    tbody: "tbody",
    td: "td",
    th: "th",
    thead: "thead",
    tr: "tr",
    ul: "ul",
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return (0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [(0,jsx_runtime.jsx)(_components.header, {
      children: (0,jsx_runtime.jsx)(_components.h1, {
        id: "api-reference",
        children: "API Reference"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "s3proxy"
      }), " exposes an S3-compatible HTTP API rather than a custom JSON API."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "This page focuses on the proxy-facing contract, supported S3 operations, and the behavior that is specific to the proxy."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "supported-operations",
      children: "Supported Operations"
    }), "\n", (0,jsx_runtime.jsxs)(_components.table, {
      children: [(0,jsx_runtime.jsx)(_components.thead, {
        children: (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.th, {
            children: "Operation"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Supported"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Notes"
          })]
        })
      }), (0,jsx_runtime.jsxs)(_components.tbody, {
        children: [(0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "GetObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Reads from one effective destination"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Reads from one effective destination"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "PutObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Can fan out with ", (0,jsx_runtime.jsx)(_components.code, {
              children: "dispatch = \"all\""
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "DeleteObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Can fan out with ", (0,jsx_runtime.jsx)(_components.code, {
              children: "dispatch = \"all\""
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadBucket"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Route-selected backend"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListObjectsV2"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Uses one effective backend only"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListObjectsV1"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Returns ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListBuckets"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Yes"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "Proxy-defined virtual bucket list"
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "CopyObject"
            })
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Returns ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: "Multipart upload operations"
          }), (0,jsx_runtime.jsx)(_components.td, {
            children: "No"
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["Return ", (0,jsx_runtime.jsx)(_components.code, {
              children: "NotImplemented"
            })]
          })]
        })]
      })]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "supported-query-keys",
      children: "Supported Query Keys"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The v1 API rejects malformed query strings with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "400 InvalidRequest"
      }), " before authentication, request-body reads, or route dispatch. Invalid/truncated percent escapes in names or values and unescaped semicolons are malformed; the proxy never executes the successfully decoded subset of such a query. Error responses and logs do not include raw query values or parser excerpts."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Well-formed queries with keys outside the supported operation contract return ", (0,jsx_runtime.jsx)(_components.code, {
        children: "501 NotImplemented"
      }), " before route dispatch, subject to normal authentication checks. For example, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?versionId=%GG"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?tagging=;value"
      }), " return 400, while ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?versionId=value"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?tagging=%3Bvalue"
      }), " are well-formed unsupported selectors and return 501. Encoded punctuation (", (0,jsx_runtime.jsx)(_components.code, {
        children: "%3B"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "%25"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "%2B"
      }), "), spaces, Unicode, and opaque continuation tokens retain normal query-decoding behavior. Existing operation-specific duplicate-key rules still apply."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Inbound SigV4 presign query keys are accepted for authentication and are not forwarded to backends: ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Algorithm"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Credential"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Date"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Expires"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Security-Token"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-Signature"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "X-Amz-SignedHeaders"
      }), "."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.table, {
      children: [(0,jsx_runtime.jsx)(_components.thead, {
        children: (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.th, {
            children: "Operation"
          }), (0,jsx_runtime.jsx)(_components.th, {
            children: "Supported non-auth query keys"
          })]
        })
      }), (0,jsx_runtime.jsxs)(_components.tbody, {
        children: [(0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsxs)(_components.td, {
            children: [(0,jsx_runtime.jsx)(_components.code, {
              children: "GetObject"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadObject"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "PutObject"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "DeleteObject"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "HeadBucket"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "ListBuckets"
            })]
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: ["none, except optional AWS SDK ", (0,jsx_runtime.jsx)(_components.code, {
              children: "x-id"
            }), " matching the operation name"]
          })]
        }), (0,jsx_runtime.jsxs)(_components.tr, {
          children: [(0,jsx_runtime.jsx)(_components.td, {
            children: (0,jsx_runtime.jsx)(_components.code, {
              children: "ListObjectsV2"
            })
          }), (0,jsx_runtime.jsxs)(_components.td, {
            children: [(0,jsx_runtime.jsx)(_components.code, {
              children: "list-type=2"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "continuation-token"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "delimiter"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "encoding-type"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "fetch-owner"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "max-keys"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "prefix"
            }), ", ", (0,jsx_runtime.jsx)(_components.code, {
              children: "start-after"
            }), ", and optional ", (0,jsx_runtime.jsx)(_components.code, {
              children: "x-id=ListObjectsV2"
            })]
          })]
        })]
      })]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Unsupported query operations include ", (0,jsx_runtime.jsx)(_components.code, {
        children: "acl"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "tagging"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "retention"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "legal-hold"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "torrent"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "versioning"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "versions"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "versionId"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "restore"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "select"
      }), ", response header overrides such as ", (0,jsx_runtime.jsx)(_components.code, {
        children: "response-content-type"
      }), ", and multipart variants such as ", (0,jsx_runtime.jsx)(_components.code, {
        children: "uploads"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "uploadId"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "partNumber"
      }), "."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "addressing-modes",
      children: "Addressing Modes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy accepts:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "path-style addressing"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "virtual-hosted addressing"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The listener decides which forms are enabled."
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "authentication-modes",
      children: "Authentication Modes"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Supported inbound auth modes:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "none"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "sigv4_static"
        })
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["With ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), ", the proxy verifies the inbound S3 SigV4 signature against statically configured clients."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Header signatures, presigned URLs, and outbound signatures use S3 canonical URI rules, preserving the escaped path without double escaping or path normalization. Spaces, percent characters, Unicode, and escaped separators in keys are supported. This corrects the previous generic-signer behavior: custom clients using the standalone AWS SDK v2 signer must set ", (0,jsx_runtime.jsx)(_components.code, {
        children: "DisableURIPathEscaping = true"
      }), ". Signatures that double-escape the path are rejected."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Header-signed requests must sign ", (0,jsx_runtime.jsx)(_components.code, {
        children: "x-amz-date"
      }), " and be within 15 minutes of the proxy clock. Presigned URLs may expire at most seven days after signing. Payload hashes may be omitted, use ", (0,jsx_runtime.jsx)(_components.code, {
        children: "UNSIGNED-PAYLOAD"
      }), ", or contain a 64-character hexadecimal SHA-256 digest. Unsupported streaming formats are rejected as described below."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "request-payload-formats",
      children: "Request Payload Formats"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Use ordinary single-request ", (0,jsx_runtime.jsx)(_components.code, {
        children: "PutObject"
      }), " uploads. The proxy does not decode AWS streaming envelopes, verify per-chunk signatures, or process S3 checksum trailers. Requests declaring any of the following return S3 ", (0,jsx_runtime.jsx)(_components.code, {
        children: "501 NotImplemented"
      }), ":"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["a comma-separated ", (0,jsx_runtime.jsx)(_components.code, {
          children: "Content-Encoding"
        }), " token equal to ", (0,jsx_runtime.jsx)(_components.code, {
          children: "aws-chunked"
        }), " (case-insensitive, with surrounding whitespace ignored)"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["an ", (0,jsx_runtime.jsx)(_components.code, {
          children: "X-Amz-Content-Sha256"
        }), " value starting with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "STREAMING-"
        }), " (case-insensitive, with surrounding whitespace ignored), including signed-payload and unsigned-trailer sentinels"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["a nonempty ", (0,jsx_runtime.jsx)(_components.code, {
          children: "X-Amz-Trailer"
        }), " value after trimming surrounding whitespace"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Header names are case-insensitive and every repeated value is checked. Encoding matching is by whole token, not substring: ", (0,jsx_runtime.jsx)(_components.code, {
        children: "not-aws-chunked"
      }), " does not trigger this boundary. The guard only inspects metadata; it neither reads nor rewrites the object body or request headers."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["This format rejection occurs after strict query validation and before authentication, payload-hash verification, replay buffering, routing, or backend I/O, in both ", (0,jsx_runtime.jsx)(_components.code, {
        children: "none"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "sigv4_static"
      }), " modes (header signatures and presigned URLs). ", (0,jsx_runtime.jsxs)(_components.strong, {
        children: ["This changes streaming hash sentinels from their former authentication failure to consistent ", (0,jsx_runtime.jsx)(_components.code, {
          children: "501 NotImplemented"
        }), "."]
      }), " Malformed queries still return ", (0,jsx_runtime.jsx)(_components.code, {
        children: "400 InvalidRequest"
      }), " first. Eligible local health probes retain their existing behavior. Errors and logs do not echo the rejected marker values."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Ordinary HTTP ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Transfer-Encoding: chunked"
      }), ", gzip-encoded object bytes with ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Content-Encoding: gzip"
      }), ", regular checksum headers, and opaque bytes that resemble AWS chunks are not rejected by this boundary. Ordinary authentication, operation, and replay limits still apply, including ", (0,jsx_runtime.jsx)(_components.code, {
        children: "413 EntityTooLarge"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "503 SlowDown"
      }), " where buffering is required. A regular checksum header is distinct from requesting unsupported trailer processing; this boundary does not itself validate checksum values."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "request-classification",
      children: "Request Classification"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routing uses S3 operation classification, not only the HTTP method."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Examples:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "GET /bucket/key"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "GetObject"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "HEAD /bucket"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "HeadBucket"
        })]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "GET /bucket?list-type=2"
        }), " can classify as ", (0,jsx_runtime.jsx)(_components.code, {
          children: "ListObjectsV2"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["That classification is what route ", (0,jsx_runtime.jsx)(_components.code, {
        children: "operations = [...]"
      }), " filters use."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "routing-specific-behavior",
      children: "Routing-Specific Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Routes are evaluated in config order and may stop or continue after a match."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Important behavior:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"first\""
        }), " uses one destination"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "dispatch = \"all\""
        }), " replays supported writes to all destinations"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "reads never fan out in v1"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "read_preference"
        }), " chooses the effective backend for reads when multiple destinations are configured"]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "streaming-responses",
      children: "Streaming Responses"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Object downloads stream from the selected backend. If copying a response fails,\nthe proxy aborts the transfer: clients receive a request or body-read error\nrather than normal completion, including for unknown-length bodies. Headers\nalready received may still show ", (0,jsx_runtime.jsx)(_components.code, {
        children: "200 OK"
      }), "; clients must read the entire body\nsuccessfully before treating the download as complete. The proxy never appends\nan XML error to an interrupted object body."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The same transfer-failure behavior applies when forwarding upstream error XML\nor transformed listing XML. A body-close error after a complete copy is logged\nas a cleanup failure and does not interrupt successful delivery. Completion\nlogs retain the committed status and bytes accepted by the response writer;\ncheck for ", (0,jsx_runtime.jsx)(_components.code, {
        children: "response copy failed"
      }), " to identify aborted transfers."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "listbuckets",
      children: (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " is virtual."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The proxy returns buckets defined in ", (0,jsx_runtime.jsx)(_components.code, {
        children: "bucket"
      }), " blocks and filtered by the authenticated client's ", (0,jsx_runtime.jsx)(_components.code, {
        children: "visible_buckets"
      }), " policy. It does not call the backend to discover buckets or resolve a route. Adding ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ListBuckets"
      }), " to a route's operations therefore has no routing effect."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "listobjectsv2",
      children: (0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      })
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "ListObjectsV2"
      }), " is forwarded to one selected backend."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy does not merge listing results or pagination tokens across multiple backends in v1."
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Bucket-only rewrites, prepend prefixes, and prefix-only templates translate\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "prefix"
      }), " and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "start-after"
      }), " to the backend namespace. XML ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Name"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Prefix"
      }), ",\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "StartAfter"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Contents.Key"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "CommonPrefixes.Prefix"
      }), " are translated back.\nSee ", (0,jsx_runtime.jsx)(_components.a, {
        href: "/docs/configuration#rewrites",
        children: "rewrite validation"
      }), " for the exact supported\ntemplate syntax; strip-key/strip-path rules are unsupported for listings."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "encoding-type=url"
      }), " is supported. Query parameters contain S3 key values after\nnormal query decoding; XML key fields are percent-decoded once, translated, and\npercent-encoded again. Encoded XML uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "+"
      }), " or ", (0,jsx_runtime.jsx)(_components.code, {
        children: "%20"
      }), " for space and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "%2B"
      }), " for a\nliteral plus; output uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "%20"
      }), " for spaces. Raw path prefixes still treat ", (0,jsx_runtime.jsx)(_components.code, {
        children: "+"
      }), "\nliterally. Keys returned by a client SDK can be used with GetObject through the same mapping.\nUse normal path escaping when constructing raw HTTP requests."]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Known query echoes (Prefix/StartAfter/Delimiter) are also accepted when exactly\nequal to the unencoded request value, as emitted by SeaweedFS; they are encoded\nconsistently in the client response. Returned keys/common prefixes must follow\nthe declared encoding."
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Continuation tokens are opaque: the proxy preserves their values without\nnamespace translation or URL decoding inside XML. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "random"
      }), " selection or\n", (0,jsx_runtime.jsx)(_components.code, {
        children: "ordered_failover"
      }), " can switch backends between pages, so tokens can be rejected\nor pages inconsistent. Choose a fixed backend for stable pagination; no cursor\naffinity or merged listing is provided. Hash-selected bucket listings are stable\nfor unchanged destinations, but object reads can hash to another replica."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Successful XML transformation is bounded to 8 MiB input/output, 50,000 elements,\nfour element levels, and five seconds, with earlier request/target cancellation\ntaking precedence. Bodies close on every exit. Malformed XML, unsupported XML\nstructure, encoding mismatches, oversized responses, or keys outside the\nrequested namespace return HTTP ", (0,jsx_runtime.jsx)(_components.code, {
        children: "502"
      }), " with S3 ", (0,jsx_runtime.jsx)(_components.code, {
        children: "InternalError"
      }), " without partial results. An object\nequal to the hidden namespace prefix has no nonempty visible key and is also\nrejected. Transformed XML gets fresh framing and integrity headers are removed;\nobject downloads continue to stream. Standard V2 owner/checksum/restore metadata\nis preserved, while unknown XML extensions fail closed."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: [(0,jsx_runtime.jsx)(_components.code, {
        children: "HeadBucket"
      }), " always addresses the rewritten bucket root, ignoring object key\nrules. It checks the backend bucket, not whether the namespace prefix exists."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "failover-rules",
      children: "Failover Rules"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "read_preference = \"ordered_failover\""
      }), ", the proxy tries the next destination on:"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "errors returned while preparing, signing, or sending the upstream request, including transport errors, request timeouts, and replay-limit errors"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["upstream ", (0,jsx_runtime.jsx)(_components.code, {
          children: "5xx"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Failover does not happen on:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "404"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchKey"
        })
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: (0,jsx_runtime.jsx)(_components.code, {
          children: "NoSuchBucket"
        })
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["any other upstream ", (0,jsx_runtime.jsx)(_components.code, {
          children: "4xx"
        })]
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "fan-out-writes",
      children: "Fan-Out Writes"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For ", (0,jsx_runtime.jsx)(_components.code, {
        children: "dispatch = \"all\""
      }), ":"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "PutObject"
        }), " is supported"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: [(0,jsx_runtime.jsx)(_components.code, {
          children: "DeleteObject"
        }), " is supported"]
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["the request body is buffered in memory so it can be replayed, bounded by ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_max_bytes"
        }), " per request and ", (0,jsx_runtime.jsx)(_components.code, {
          children: "listener.replay_body_aggregate_max_bytes"
        }), " across the process"]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "if any destination fails, the request fails overall"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "upstream HTTP failures preserve the primary upstream error response when available"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "transport or replay failures return a proxy-generated failure"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "fan-out is not transactional; a destination that succeeds before another destination fails is not rolled back"
      }), "\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["oversized replay attempts fail with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "413 EntityTooLarge"
        }), "; aggregate replay-budget exhaustion fails with ", (0,jsx_runtime.jsx)(_components.code, {
          children: "503 SlowDown"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "at most four destination attempts run concurrently; additional destinations wait for a slot"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For writes matched by multiple routes through ", (0,jsx_runtime.jsx)(_components.code, {
        children: "on_match = \"continue\""
      }), ", every matched route must also succeed. A later route failure is returned as failure rather than hiding behind an earlier success."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "outbound-signing",
      children: "Outbound Signing"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "The proxy terminates and rebuilds requests before forwarding them."
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Outbound S3 requests are signed with the destination backend credentials, not the inbound client credentials."
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["For outbound SigV4, the proxy uses ", (0,jsx_runtime.jsx)(_components.code, {
        children: "UNSIGNED-PAYLOAD"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Content-Length"
      }), " must be set before signing."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h3, {
      id: "upstream-redirects",
      children: "Upstream Redirects"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Upstream ", (0,jsx_runtime.jsx)(_components.code, {
        children: "301"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "302"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "303"
      }), ", ", (0,jsx_runtime.jsx)(_components.code, {
        children: "307"
      }), ", and ", (0,jsx_runtime.jsx)(_components.code, {
        children: "308"
      }), " responses are rejected for every\noperation, including object reads, uploads, deletes and listings. The proxy\ndoes not follow the redirect or forward its ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Location"
      }), " or response body. The\nfailure uses HTTP ", (0,jsx_runtime.jsx)(_components.code, {
        children: "502"
      }), " with S3 ", (0,jsx_runtime.jsx)(_components.code, {
        children: "InternalError"
      }), " through ordinary proxy error\nhandling. Configured ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ordered_failover"
      }), " may instead succeed at the next\nconfigured destination; fan-out preserves an available primary non-redirect\nHTTP error response under its normal rules."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Configure the correct backend endpoint and region. Redirect-based region\ndiscovery is unsupported. Redirect URL credentials, object paths and query\nvalues are excluded from errors and logs. Conditional ", (0,jsx_runtime.jsx)(_components.code, {
        children: "304 Not Modified"
      }), " and\nordinary upstream HTTP error responses are unaffected."]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "error-behavior",
      children: "Error Behavior"
    }), "\n", (0,jsx_runtime.jsx)(_components.p, {
      children: "Representative error rules:"
    }), "\n", (0,jsx_runtime.jsxs)(_components.ul, {
      children: ["\n", (0,jsx_runtime.jsxs)(_components.li, {
        children: ["unsupported operations return S3-compatible ", (0,jsx_runtime.jsx)(_components.code, {
          children: "NotImplemented"
        })]
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "route misses return standard S3-compatible error responses"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "upstream backend failures propagate as proxy-mediated S3 responses"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "multi-destination write failures are surfaced as failures, not partial success"
      }), "\n", (0,jsx_runtime.jsx)(_components.li, {
        children: "multi-route write failures are surfaced as failures, not partial success"
      }), "\n"]
    }), "\n", (0,jsx_runtime.jsx)(_components.h2, {
      id: "health-endpoints",
      children: "Health Endpoints"
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["Local probes on the main S3 listener return ", (0,jsx_runtime.jsx)(_components.code, {
        children: "200 OK"
      }), " with body ", (0,jsx_runtime.jsx)(_components.code, {
        children: "ok"
      }), " only for ", (0,jsx_runtime.jsx)(_components.strong, {
        children: "GET"
      }), " requests with the exact escaped path ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/healthz"
      }), " or ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/readyz"
      }), ", no ", (0,jsx_runtime.jsx)(_components.code, {
        children: "Authorization"
      }), " header (even an empty one), no query string (including a bare ", (0,jsx_runtime.jsx)(_components.code, {
        children: "?"
      }), "), and a host that is not a configured virtual bucket host. Ordinary localhost, IP, and base-host probes work even on virtual-host-only listeners. ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/readyz"
      }), " means the proxy process is serving requests; it does not poll configured backends or report destination health."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["All other requests follow the normal S3 parsing, authentication, authorization, and operation handling. This includes HEAD, signed or presigned requests, query-bearing requests, encoded spellings such as ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/%68ealthz"
      }), ", and requests to configured virtual bucket hosts. These paths can therefore be object keys on virtual bucket hosts, or path-style bucket names for ListObjectsV2 and HEAD. Malformed probe-path queries (for example, ", (0,jsx_runtime.jsx)(_components.code, {
        children: "/healthz?prefix=%GG"
      }), ") return ", (0,jsx_runtime.jsx)(_components.code, {
        children: "400 InvalidRequest"
      }), ". Unsupported operations retain their normal S3 errors; there is no probe-specific 405 response."]
    }), "\n", (0,jsx_runtime.jsxs)(_components.p, {
      children: ["The indistinguishable unsigned, exact, query-free base-host GET shape is reserved for probes. Virtual bucket recognition uses the listener's enabled virtual-host addressing and ordered ", (0,jsx_runtime.jsx)(_components.code, {
        children: "host_suffixes"
      }), ", including case, port, and trailing-dot normalization. Unconfigured host aliases cannot be inferred to be bucket hosts and may receive the local probe response. Configure probe hosts accordingly. Restrict probe access at the network or reverse-proxy layer if the listener is public. This GET-only contract replaces the former method-agnostic behavior; signed requests are no longer probes."]
    })]
  });
}
function MDXContent(props = {}) {
  const {wrapper: MDXLayout} = {
    ...(0,lib/* useMDXComponents */.R)(),
    ...props.components
  };
  return MDXLayout ? (0,jsx_runtime.jsx)(MDXLayout, {
    ...props,
    children: (0,jsx_runtime.jsx)(_createMdxContent, {
      ...props
    })
  }) : _createMdxContent(props);
}



/***/ },

/***/ 1982
(__unused_webpack___webpack_module__, __webpack_exports__, __webpack_require__) {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   R: () => (/* binding */ useMDXComponents),
/* harmony export */   x: () => (/* binding */ MDXProvider)
/* harmony export */ });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(489);
/**
 * @import {MDXComponents} from 'mdx/types.js'
 * @import {Component, ReactElement, ReactNode} from 'react'
 */

/**
 * @callback MergeComponents
 *   Custom merge function.
 * @param {Readonly<MDXComponents>} currentComponents
 *   Current components from the context.
 * @returns {MDXComponents}
 *   Additional components.
 *
 * @typedef Props
 *   Configuration for `MDXProvider`.
 * @property {ReactNode | null | undefined} [children]
 *   Children (optional).
 * @property {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @property {boolean | null | undefined} [disableParentContext=false]
 *   Turn off outer component context (default: `false`).
 */



/** @type {Readonly<MDXComponents>} */
const emptyComponents = {}

const MDXContext = react__WEBPACK_IMPORTED_MODULE_0__.createContext(emptyComponents)

/**
 * Get current components from the MDX Context.
 *
 * @param {Readonly<MDXComponents> | MergeComponents | null | undefined} [components]
 *   Additional components to use or a function that creates them (optional).
 * @returns {MDXComponents}
 *   Current components.
 */
function useMDXComponents(components) {
  const contextComponents = react__WEBPACK_IMPORTED_MODULE_0__.useContext(MDXContext)

  // Memoize to avoid unnecessary top-level context changes
  return react__WEBPACK_IMPORTED_MODULE_0__.useMemo(
    function () {
      // Custom merge via a function prop
      if (typeof components === 'function') {
        return components(contextComponents)
      }

      return {...contextComponents, ...components}
    },
    [contextComponents, components]
  )
}

/**
 * Provider for MDX context.
 *
 * @param {Readonly<Props>} properties
 *   Properties.
 * @returns {ReactElement}
 *   Element.
 * @satisfies {Component}
 */
function MDXProvider(properties) {
  /** @type {Readonly<MDXComponents>} */
  let allComponents

  if (properties.disableParentContext) {
    allComponents =
      typeof properties.components === 'function'
        ? properties.components(emptyComponents)
        : properties.components || emptyComponents
  } else {
    allComponents = useMDXComponents(properties.components)
  }

  return react__WEBPACK_IMPORTED_MODULE_0__.createElement(
    MDXContext.Provider,
    {value: allComponents},
    properties.children
  )
}


/***/ }

}]);