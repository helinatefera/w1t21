import { useParams, Link } from 'react-router-dom';
import { useState, useRef, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listMessages, sendMessage } from '../api/messages';
import { useAuthStore } from '../store/authStore';
import { detectPII } from '../utils/piiDetector';
import { formatRelativeTime } from '../utils/formatters';

const MAX_ATTACHMENT_SIZE = 10 * 1024 * 1024;

export function MessagesPage() {
  const { id: orderId } = useParams<{ id: string }>();
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const [body, setBody] = useState('');
  const [attachment, setAttachment] = useState<File | null>(null);
  const [piiWarning, setPiiWarning] = useState('');
  const [fileError, setFileError] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { data } = useQuery({
    queryKey: ['messages', orderId],
    queryFn: () => listMessages(orderId!, 1, 100).then((r) => r.data),
    enabled: !!orderId,
    refetchInterval: 10000,
  });

  const [sendError, setSendError] = useState('');

  const mutation = useMutation({
    mutationFn: () => sendMessage(orderId!, body, attachment || undefined),
    onSuccess: () => {
      setBody('');
      setAttachment(null);
      setPiiWarning('');
      setSendError('');
      queryClient.invalidateQueries({ queryKey: ['messages', orderId] });
    },
    onError: (err: any) => {
      const status = err.response?.status;
      const errorData = err.response?.data?.error;
      const code = errorData?.code;
      const message = errorData?.message;

      if (status === 429 || code === 'ERR_RATE_LIMITED') {
        setSendError('Message blocked: You are sending messages too quickly. Please wait a moment before trying again.');
      } else if (status === 403 || code === 'ERR_FORBIDDEN') {
        setSendError('Message blocked: You are not authorized to send messages on this order. Only the buyer and seller can communicate.');
      } else if (status === 413 || code === 'ERR_ATTACHMENT_TOO_LARGE') {
        setSendError('Message blocked: The attachment exceeds the 10MB size limit. Please use a smaller file.');
      } else if (status === 422 || code === 'ERR_VALIDATION') {
        setSendError(`Message blocked: ${message || 'Invalid message content. Please check your input.'}`);
      } else {
        setSendError(message || 'Failed to send message. Please try again.');
      }
    },
  });

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [data]);

  const handleSend = () => {
    const piiResult = detectPII(body);
    if (piiResult.detected) {
      setPiiWarning(`Message blocked: Your message contains sensitive personal information (${piiResult.types.join(', ')}). Please remove it before sending to protect privacy.`);
      return;
    }
    setPiiWarning('');
    setSendError('');
    mutation.mutate();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    setFileError('');
    if (file) {
      if (file.size > MAX_ATTACHMENT_SIZE) {
        setFileError('Attachment exceeds 10MB limit.');
        setAttachment(null);
        return;
      }
      setAttachment(file);

      // Client-side PII pre-check: scan all readable file content.
      // Text files are read as text; PDFs and SVGs are read as text
      // too since they contain extractable strings. Binary images
      // are read as text — regex PII patterns won't match random
      // bytes, so they pass through harmlessly. The server performs
      // the same scan as the authoritative enforcement layer.
      const reader = new FileReader();
      reader.onload = (ev) => {
        const content = ev.target?.result as string;
        const piiResult = detectPII(content);
        if (piiResult.detected) {
          setFileError(`Attachment contains sensitive content (${piiResult.types.join(', ')}). Remove PII before sending.`);
          setAttachment(null);
        }
      };
      reader.readAsText(file);
    }
  };

  return (
    <div className="max-w-3xl mx-auto">
      <Link to={`/orders/${orderId}`} className="text-sm text-primary-600 hover:underline mb-4 block">
        Back to Order
      </Link>

      <div className="bg-white rounded-lg shadow-sm border flex flex-col h-[500px] md:h-[600px] lg:h-[700px]">
        <div className="p-4 border-b">
          <h2 className="font-semibold">Messages - Order #{orderId?.slice(0, 8)}</h2>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {data?.data?.map((msg) => {
            const isMe = msg.sender_id === user?.id;
            return (
              <div key={msg.id} className={`flex ${isMe ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[70%] p-3 rounded-lg ${isMe ? 'bg-primary-100 text-primary-900' : 'bg-gray-100'}`}>
                  <p className="text-sm">{msg.body}</p>
                  {msg.attachment_id && (
                    <a
                      href={`/api/messages/${msg.attachment_id}/attachment`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-xs mt-1 text-primary-600 hover:underline block"
                    >
                      Attachment ({msg.attachment_mime}, {Math.round((msg.attachment_size || 0) / 1024)}KB)
                    </a>
                  )}
                  <p className="text-xs text-gray-400 mt-1">{formatRelativeTime(msg.created_at)}</p>
                </div>
              </div>
            );
          })}
          <div ref={messagesEndRef} />
        </div>

        <div className="p-4 border-t">
          {piiWarning && (
            <div className="text-sm text-red-600 bg-red-50 p-3 rounded mb-2 border border-red-200">
              <span className="font-medium">Blocked:</span> {piiWarning}
            </div>
          )}
          {sendError && (
            <div className="text-sm text-red-600 bg-red-50 p-3 rounded mb-2 border border-red-200">
              <span className="font-medium">Error:</span> {sendError}
            </div>
          )}
          {fileError && (
            <div className="text-sm text-red-600 bg-red-50 p-3 rounded mb-2 border border-red-200">
              <span className="font-medium">File Error:</span> {fileError}
            </div>
          )}
          <div className="flex gap-2">
            <input
              type="text"
              value={body}
              onChange={(e) => { setBody(e.target.value); setPiiWarning(''); }}
              placeholder="Type a message..."
              className="flex-1 px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary-500"
              onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && handleSend()}
            />
            <label className="px-3 py-2 bg-gray-100 rounded cursor-pointer text-sm hover:bg-gray-200">
              Attach
              <input type="file" className="hidden" onChange={handleFileChange} />
            </label>
            <button
              onClick={handleSend}
              disabled={!body.trim() || mutation.isPending}
              className="px-4 py-2 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 disabled:opacity-50"
            >
              Send
            </button>
          </div>
          {attachment && (
            <p className="text-xs text-gray-500 mt-1">
              Attached: {attachment.name} ({Math.round(attachment.size / 1024)}KB)
              <button onClick={() => setAttachment(null)} className="ml-2 text-red-500">Remove</button>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
